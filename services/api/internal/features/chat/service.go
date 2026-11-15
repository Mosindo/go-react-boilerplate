package chat

import (
	"context"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/apperr"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/textx"
)

var (
	ErrConversationNotFound = apperr.NotFound("conversation_not_found", "conversation not found")
	ErrEmptyMessage         = apperr.BadRequest("empty_message", "message cannot be empty")
	ErrMessageTooLong       = apperr.BadRequest("message_too_long", "message must be at most 2000 characters")
)

type Service struct {
	repo          Repository
	cards         *profiles.Service
	notifications *notifications.Service
	publisher     realtime.Publisher
}

func NewService(repo Repository, cards *profiles.Service, notifier *notifications.Service, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, notifications: notifier, publisher: publisher}
}

func (s *Service) ListConversations(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) (ConversationsResponse, error) {
	rows, err := s.repo.ListConversations(ctx, userID, cursor, limit+1, "")
	if err != nil {
		return ConversationsResponse{}, err
	}
	var next *string
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[limit-1]
		encoded := httpx.EncodeCursor(last.SortAt, last.ID)
		next = &encoded
	}
	items, err := s.decorate(ctx, userID, rows)
	if err != nil {
		return ConversationsResponse{}, err
	}
	return ConversationsResponse{Conversations: items, NextCursor: next}, nil
}

func (s *Service) GetConversation(ctx context.Context, userID, conversationID string) (ConversationResponse, error) {
	if _, err := s.repo.Membership(ctx, userID, conversationID); err != nil {
		return ConversationResponse{}, err
	}
	rows, err := s.repo.ListConversations(ctx, userID, nil, 1, conversationID)
	if err != nil {
		return ConversationResponse{}, err
	}
	items, err := s.decorate(ctx, userID, rows)
	if err != nil {
		return ConversationResponse{}, err
	}
	if len(items) == 0 {
		// Cleared locally and no new message: still reachable directly.
		m, err := s.repo.Membership(ctx, userID, conversationID)
		if err != nil {
			return ConversationResponse{}, err
		}
		cards, err := s.cards.LoadCards(ctx, userID, []string{m.OtherUserID})
		if err != nil {
			return ConversationResponse{}, err
		}
		return ConversationResponse{ID: conversationID, User: cards[m.OtherUserID], OtherLastReadAt: m.OtherLastReadAt}, nil
	}
	return items[0], nil
}

func (s *Service) decorate(ctx context.Context, userID string, rows []ConversationRow) ([]ConversationResponse, error) {
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.OtherUserID
	}
	cards, err := s.cards.LoadCards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]ConversationResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, ConversationResponse{
			ID:              r.ID,
			MatchID:         r.MatchID,
			MatchedAt:       r.MatchedAt,
			User:            cards[r.OtherUserID],
			LastMessage:     r.LastMessage,
			UnreadCount:     r.UnreadCount,
			OtherLastReadAt: r.OtherLastReadAt,
		})
	}
	return out, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID string, cursor *httpx.Cursor, limit int) (MessagesResponse, error) {
	m, err := s.repo.Membership(ctx, userID, conversationID)
	if err != nil {
		return MessagesResponse{}, err
	}
	msgs, err := s.repo.ListMessages(ctx, conversationID, m.ClearedAt, cursor, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	var next *string
	if len(msgs) > limit {
		msgs = msgs[:limit]
		last := msgs[limit-1]
		encoded := httpx.EncodeCursor(last.CreatedAt, last.ID)
		next = &encoded
	}
	return MessagesResponse{Messages: msgs, NextCursor: next, OtherLastReadAt: m.OtherLastReadAt}, nil
}

// ValidateMessage normalises a message body.
func ValidateMessage(body string) (string, error) {
	cleaned := textx.Clean(body, true)
	if cleaned == "" {
		return "", ErrEmptyMessage
	}
	if textx.Len(cleaned) > MaxMessageRunes {
		return "", ErrMessageTooLong
	}
	return cleaned, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	cleaned, err := ValidateMessage(body)
	if err != nil {
		return Message{}, err
	}
	m, err := s.repo.Membership(ctx, userID, conversationID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.CreateMessage(ctx, userID, conversationID, cleaned)
	if err != nil {
		return Message{}, err
	}

	if s.publisher != nil {
		event := realtime.Event{Type: "message.new", Data: msg}
		s.publisher.Publish(m.OtherUserID, event)
		s.publisher.Publish(userID, event)
	}

	senderName := "Nouveau message"
	if cards, err := s.cards.LoadCards(ctx, m.OtherUserID, []string{userID}); err == nil {
		if card, ok := cards[userID]; ok && card.FirstName != "" {
			senderName = card.FirstName
		}
	}
	s.notifications.Notify(ctx, notifications.Input{
		UserID:   m.OtherUserID,
		Type:     notifications.TypeMessage,
		Title:    senderName,
		Body:     preview(cleaned),
		Data:     map[string]string{"conversationId": conversationID, "userId": userID},
		GroupKey: "conversation:" + conversationID,
	})
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (time.Time, error) {
	m, err := s.repo.Membership(ctx, userID, conversationID)
	if err != nil {
		return time.Time{}, err
	}
	at, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		return time.Time{}, err
	}
	s.notifications.MarkGroupRead(ctx, userID, "conversation:"+conversationID)
	if s.publisher != nil {
		s.publisher.Publish(m.OtherUserID, realtime.Event{Type: "conversation.read", Data: map[string]any{
			"conversationId": conversationID, "userId": userID, "readAt": at,
		}})
	}
	return at, nil
}

func (s *Service) Clear(ctx context.Context, userID, conversationID string) error {
	if err := s.repo.Clear(ctx, userID, conversationID); err != nil {
		return err
	}
	s.notifications.MarkGroupRead(ctx, userID, "conversation:"+conversationID)
	return nil
}

func preview(body string) string {
	runes := []rune(body)
	if len(runes) > 120 {
		return string(runes[:117]) + "…"
	}
	return body
}
