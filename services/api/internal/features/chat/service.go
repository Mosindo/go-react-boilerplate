package chat

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

var ErrInvalidMessage = errors.New("chat: invalid message")

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) ListConversations(ctx context.Context, userID string, limit, offset int) ([]Conversation, error) {
	return s.repo.ListConversations(ctx, userID, limit, offset)
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID, beforeID string, limit int) ([]Message, bool, error) {
	return s.repo.ListMessages(ctx, userID, conversationID, beforeID, limit)
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	clean := sanitizeBody(body)
	if clean == "" || utf8.RuneCountInString(clean) > MaxMessageRunes {
		return Message{}, ErrInvalidMessage
	}
	sent, err := s.repo.SendMessage(ctx, userID, conversationID, clean)
	if err != nil {
		return Message{}, err
	}
	event := realtime.Event{Type: "message.new", Data: sent.Message}
	s.pub.Publish(sent.RecipientID, event)
	s.pub.Publish(userID, event) // the sender's other devices
	s.pub.Publish(sent.RecipientID, realtime.Event{Type: "notification.new", Data: sent.Notification})
	return sent.Message, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	rc, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	s.pub.Publish(rc.OtherUserID, realtime.Event{Type: "message.read", Data: map[string]any{
		"conversationId": conversationID, "readerId": userID, "readAt": rc.ReadAt,
	}})
	return nil
}

func (s *Service) Clear(ctx context.Context, userID, conversationID string) error {
	return s.repo.Clear(ctx, userID, conversationID)
}

// sanitizeBody trims, normalises line endings and drops control characters other than newlines.
func sanitizeBody(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, body)
	return strings.TrimSpace(body)
}
