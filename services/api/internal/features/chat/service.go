package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrEmptyMessage         = errors.New("message cannot be empty")
	ErrMessageTooLong       = errors.New("message cannot exceed 2000 characters")
	ErrInvalidFilter        = errors.New("filter must be one of all, new, active")
)

type CardLoader interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Card, error)
}

type MessageNotifier interface {
	NotifyMessage(ctx context.Context, userID, senderFirstName, conversationID, preview string)
	MarkConversationRead(ctx context.Context, userID, conversationID string)
}

type Service struct {
	repo      Repository
	cards     CardLoader
	notifier  MessageNotifier
	publisher realtime.Publisher
}

func NewService(repo Repository, cards CardLoader, notifier MessageNotifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, notifier: notifier, publisher: publisher}
}

// NormalizeMessageBody trims the message, removes control characters other
// than new lines and tabs, and enforces the length limit.
func NormalizeMessageBody(body string) (string, error) {
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, body)
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(cleaned) > maxMessageRunes {
		return "", ErrMessageTooLong
	}
	return cleaned, nil
}

func (s *Service) List(ctx context.Context, userID, filter string, limit, offset int) ([]Conversation, error) {
	if filter == "" {
		filter = FilterAll
	}
	if filter != FilterAll && filter != FilterNew && filter != FilterActive {
		return nil, ErrInvalidFilter
	}
	return s.list(ctx, userID, filter, nil, limit, offset)
}

func (s *Service) list(ctx context.Context, userID, filter string, onlyID *string, limit, offset int) ([]Conversation, error) {
	list, err := s.repo.List(ctx, userID, filter, onlyID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(list))
	for _, c := range list {
		ids = append(ids, c.OtherUserID)
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Conversation, 0, len(list))
	for _, c := range list {
		card, ok := cards[c.OtherUserID]
		if !ok {
			continue
		}
		c.Other = profiles.ToCardSummary(card)
		out = append(out, c)
	}
	return out, nil
}

func (s *Service) membership(ctx context.Context, conversationID, userID string) (Membership, error) {
	m, err := s.repo.Membership(ctx, conversationID, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Membership{}, ErrConversationNotFound
		}
		return Membership{}, err
	}
	return m, nil
}

// Get returns one conversation header (used when opening a thread).
func (s *Service) Get(ctx context.Context, userID, conversationID string) (Conversation, error) {
	if _, err := s.membership(ctx, conversationID, userID); err != nil {
		return Conversation{}, err
	}
	// The "all" filter hides locally cleared threads, so query without it.
	list, err := s.list(ctx, userID, "any", &conversationID, 1, 0)
	if err != nil {
		return Conversation{}, err
	}
	if len(list) == 0 {
		return Conversation{}, ErrConversationNotFound
	}
	return list[0], nil
}

func (s *Service) Messages(ctx context.Context, userID, conversationID string, beforeID *string, limit int) ([]Message, bool, *time.Time, error) {
	m, err := s.membership(ctx, conversationID, userID)
	if err != nil {
		return nil, false, nil, err
	}
	messages, err := s.repo.Messages(ctx, conversationID, m.ClearedAt, beforeID, limit+1)
	if err != nil {
		return nil, false, nil, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	return messages, hasMore, m.OtherLastReadAt, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	normalized, err := NormalizeMessageBody(body)
	if err != nil {
		return Message{}, err
	}
	m, err := s.membership(ctx, conversationID, userID)
	if err != nil {
		return Message{}, err
	}
	msg, senderName, err := s.repo.CreateMessage(ctx, conversationID, userID, normalized)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Message{}, ErrConversationNotFound
		}
		return Message{}, err
	}
	event := realtime.Event{Type: "message.created", Data: toMessageResponse(msg)}
	s.publisher.Publish(userID, event)
	s.publisher.Publish(m.OtherUserID, event)
	s.notifier.NotifyMessage(ctx, m.OtherUserID, senderName, conversationID, normalized)
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (*time.Time, error) {
	m, err := s.membership(ctx, conversationID, userID)
	if err != nil {
		return nil, err
	}
	lastRead, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	s.notifier.MarkConversationRead(ctx, userID, conversationID)
	if lastRead != nil {
		s.publisher.Publish(m.OtherUserID, realtime.Event{Type: "conversation.read", Data: ReadEvent{
			ConversationID: conversationID, ReaderID: userID, LastReadAt: *lastRead,
		}})
	}
	return lastRead, nil
}

// Clear hides the current history for the caller only ("delete conversation
// for me"). The other member keeps their copy; new messages appear again.
func (s *Service) Clear(ctx context.Context, userID, conversationID string) error {
	if _, err := s.membership(ctx, conversationID, userID); err != nil {
		return err
	}
	if err := s.repo.Clear(ctx, conversationID, userID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrConversationNotFound
		}
		return err
	}
	s.notifier.MarkConversationRead(ctx, userID, conversationID)
	return nil
}
