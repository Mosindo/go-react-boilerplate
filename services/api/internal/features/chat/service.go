package chat

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo     Repository
	realtime realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, realtime: publisher}
}

func (s *Service) ListConversations(ctx context.Context, userID string, before *time.Time, limit int) (ConversationsResponse, error) {
	limit = clamp(limit, DefaultConversations, MaxConversations)
	list, err := s.repo.ListConversations(ctx, userID, before, limit)
	if err != nil {
		return ConversationsResponse{}, err
	}
	resp := ConversationsResponse{Conversations: list}
	if len(list) == limit {
		last := list[len(list)-1].LastMessageAt
		resp.NextCursor = &last
	}
	return resp, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID string, before *time.Time, limit int) (MessagesResponse, error) {
	if !profiles.IsUUID(conversationID) {
		return MessagesResponse{}, ErrConversationNotFound
	}
	limit = clamp(limit, DefaultMessagesLimit, MaxMessagesLimit)
	list, err := s.repo.ListMessages(ctx, userID, conversationID, before, limit)
	if err != nil {
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{Messages: list}
	if len(list) == limit {
		oldest := list[0].CreatedAt
		resp.NextCursor = &oldest
	}
	return resp, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, content string) (Message, error) {
	if !profiles.IsUUID(conversationID) {
		return Message{}, ErrConversationNotFound
	}
	text := cleanMessage(content)
	if text == "" {
		return Message{}, platformerrors.Invalid("content", "le message est vide")
	}
	if utf8.RuneCountInString(text) > MaxMessageRunes {
		return Message{}, platformerrors.Invalid("content", "le message ne peut pas dépasser 2000 caractères")
	}

	msg, recipientID, notified, err := s.repo.CreateMessage(ctx, userID, conversationID, text)
	if err != nil {
		return Message{}, err
	}
	event := realtime.Event{Type: "message.new", Data: msg}
	s.realtime.Publish(recipientID, event)
	s.realtime.Publish(userID, event) // the sender's other devices
	if notified {
		s.realtime.Publish(recipientID, realtime.Event{Type: "notification.new", Data: map[string]string{
			"type": "message", "conversationId": conversationID,
		}})
	}
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (int, error) {
	if !profiles.IsUUID(conversationID) {
		return 0, ErrConversationNotFound
	}
	peerID, count, readAt, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		return 0, err
	}
	if count > 0 {
		s.realtime.Publish(peerID, realtime.Event{Type: "message.read", Data: map[string]any{
			"conversationId": conversationID, "readAt": readAt, "readerId": userID,
		}})
	}
	return count, nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	if !profiles.IsUUID(conversationID) {
		return ErrConversationNotFound
	}
	return s.repo.Hide(ctx, userID, conversationID)
}

// cleanMessage trims the text and removes control characters other than newlines.
func cleanMessage(v string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(v) {
		if unicode.IsControl(r) && r != '\n' {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func clamp(v, def, max int) int {
	if v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}
