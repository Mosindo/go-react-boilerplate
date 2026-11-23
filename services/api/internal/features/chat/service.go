package chat

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

var (
	ErrEmptyMessage   = errors.New("message cannot be empty")
	ErrMessageTooLong = errors.New("message must be at most 2000 characters")
)

type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string)
	MarkConversationRead(ctx context.Context, userID, conversationID string)
}

// NameResolver gives the sender's first name for notification copy.
type NameResolver interface {
	FirstName(ctx context.Context, userID string) string
}

type Service struct {
	repo      Repository
	publisher realtime.Publisher
	notifier  Notifier
	names     NameResolver
}

func NewService(repo Repository, publisher realtime.Publisher, notifier Notifier, names NameResolver) *Service {
	return &Service{repo: repo, publisher: publisher, notifier: notifier, names: names}
}

// NormalizeContent trims a message and strips control characters other than
// newlines and tabs.
func NormalizeContent(raw string) (string, error) {
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, raw)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(cleaned) > MaxMessageRunes {
		return "", ErrMessageTooLong
	}
	return cleaned, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID, beforeID string, limit int) (MessagesResponse, error) {
	membership, err := s.repo.GetMembership(ctx, userID, conversationID)
	if err != nil {
		return MessagesResponse{}, err
	}
	messages, err := s.repo.ListMessages(ctx, conversationID, beforeID, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	return MessagesResponse{
		Messages:        messages,
		HasMore:         hasMore,
		OtherUserID:     membership.OtherUserID,
		OtherLastReadAt: membership.OtherLastReadAt,
	}, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, content string) (Message, error) {
	normalized, err := NormalizeContent(content)
	if err != nil {
		return Message{}, err
	}
	membership, err := s.repo.GetMembership(ctx, userID, conversationID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.CreateMessage(ctx, conversationID, userID, membership.OtherUserID, normalized)
	if err != nil {
		return Message{}, err
	}

	if s.publisher != nil {
		s.publisher.Publish([]string{userID, membership.OtherUserID}, realtime.Event{Type: "message.created", Data: msg})
	}
	if s.notifier != nil {
		name := "Quelqu'un"
		if s.names != nil {
			if n := s.names.FirstName(ctx, userID); n != "" {
				name = n
			}
		}
		s.notifier.Notify(ctx, membership.OtherUserID, "message", "Nouveau message", name+" t'a écrit.", map[string]string{
			"conversationId": conversationID,
			"userId":         userID,
			"collapseKey":    "conversation:" + conversationID,
		})
	}
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (ReadReceipt, error) {
	membership, err := s.repo.GetMembership(ctx, userID, conversationID)
	if err != nil {
		return ReadReceipt{}, err
	}
	readAt, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		return ReadReceipt{}, err
	}
	receipt := ReadReceipt{ConversationID: conversationID, UserID: userID, LastReadAt: readAt}
	if s.notifier != nil {
		s.notifier.MarkConversationRead(ctx, userID, conversationID)
	}
	if s.publisher != nil {
		s.publisher.Publish([]string{membership.OtherUserID, userID}, realtime.Event{Type: "conversation.read", Data: receipt})
	}
	return receipt, nil
}
