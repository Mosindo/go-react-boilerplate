package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/photos"
	"example.com/api/internal/features/profiles"
	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var (
	// A conversation the caller does not belong to is reported as missing,
	// so its existence is never disclosed.
	ErrConversationNotFound = apperr.NotFound("conversation_not_found", "conversation not found")
	ErrEmptyMessage         = apperr.BadRequest("empty_message", "message cannot be empty")
	ErrMessageTooLong       = apperr.BadRequest("message_too_long", "message must be 2000 characters or less")
)

type PhotoSource interface {
	ListForUsers(ctx context.Context, userIDs []string) (map[string][]photos.PhotoResponse, error)
}

type Service struct {
	repo      Repository
	photos    PhotoSource
	publisher realtime.Publisher
	now       func() time.Time
}

func NewService(repo Repository, photoSource PhotoSource, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, photos: photoSource, publisher: publisher, now: time.Now}
}

// NormalizeMessage trims the body and strips control characters.
func NormalizeMessage(body string) (string, error) {
	cleaned := strings.TrimSpace(strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, body))
	if cleaned == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(cleaned) > MaxMessageRunes {
		return "", ErrMessageTooLong
	}
	return cleaned, nil
}

func (s *Service) ListConversations(ctx context.Context, userID, beforeID string, limit int) (ConversationsResponse, error) {
	items, err := s.repo.ListConversations(ctx, userID, beforeID, limit+1)
	if err != nil {
		return ConversationsResponse{}, err
	}
	resp := ConversationsResponse{Conversations: items}
	if len(items) > limit {
		resp.Conversations = items[:limit]
		cursor := items[limit-1].ID
		resp.NextCursor = &cursor
	}
	ids := make([]string, 0, len(resp.Conversations))
	for _, c := range resp.Conversations {
		ids = append(ids, c.OtherUser.UserID)
	}
	photoMap, err := s.photos.ListForUsers(ctx, ids)
	if err != nil {
		return ConversationsResponse{}, err
	}
	now := s.now()
	for i := range resp.Conversations {
		c := &resp.Conversations[i]
		c.OtherUser.Age = profiles.AgeAt(c.otherBirthday, now)
		if list := photoMap[c.OtherUser.UserID]; len(list) > 0 {
			primary := list[0]
			c.OtherUser.Photo = &primary
		}
	}
	return resp, nil
}

func (s *Service) membership(ctx context.Context, userID, conversationID string) (Membership, error) {
	m, err := s.repo.Membership(ctx, userID, conversationID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Membership{}, ErrConversationNotFound
		}
		return Membership{}, err
	}
	return m, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID, beforeID string, limit int) (MessagesResponse, error) {
	m, err := s.membership(ctx, userID, conversationID)
	if err != nil {
		return MessagesResponse{}, err
	}
	items, err := s.repo.ListMessages(ctx, m, beforeID, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{Messages: items, OtherLastReadAt: m.OtherLastReadAt}
	if len(items) > limit {
		resp.Messages = items[:limit]
		cursor := items[limit-1].ID
		resp.NextCursor = &cursor
	}
	for i := range resp.Messages {
		msg := &resp.Messages[i]
		if msg.SenderID == userID {
			msg.IsRead = m.OtherLastReadAt != nil && !m.OtherLastReadAt.Before(msg.CreatedAt)
		} else {
			msg.IsRead = m.MyLastReadAt != nil && !m.MyLastReadAt.Before(msg.CreatedAt)
		}
	}
	return resp, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	cleaned, err := NormalizeMessage(body)
	if err != nil {
		return Message{}, err
	}
	m, err := s.membership(ctx, userID, conversationID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.CreateMessage(ctx, userID, conversationID, cleaned)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Message{}, ErrConversationNotFound
		}
		return Message{}, err
	}
	event := realtime.Event{Type: "message.created", Data: msg}
	s.publisher.Publish(m.OtherUserID, event)
	s.publisher.Publish(userID, event)
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	m, err := s.membership(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	readAt, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrConversationNotFound
		}
		return err
	}
	s.publisher.Publish(m.OtherUserID, realtime.Event{Type: "conversation.read", Data: map[string]any{
		"conversationId": conversationID,
		"userId":         userID,
		"lastReadAt":     readAt,
	}})
	return nil
}

// Hide removes the conversation from the caller's list only. It reappears if
// a new message arrives; older messages stay hidden for the caller.
func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	if err := s.repo.Hide(ctx, userID, conversationID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrConversationNotFound
		}
		return err
	}
	return nil
}
