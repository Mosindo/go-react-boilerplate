package chat

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

// CardLoader is implemented by profiles.Service.
type CardLoader interface {
	Cards(ctx context.Context, viewerID string, ids []string) (map[string]profiles.Card, error)
}

type Service struct {
	repo  Repository
	cards CardLoader
	pub   realtime.Publisher
}

func NewService(repo Repository, cards CardLoader, pub realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, pub: pub}
}

func (s *Service) ListConversations(ctx context.Context, userID string, limit, offset int) ([]Conversation, error) {
	rows, err := s.repo.ListConversations(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.OtherUserID
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Conversation, 0, len(rows))
	for _, r := range rows {
		card, ok := cards[r.OtherUserID]
		if !ok {
			continue // counterpart suspended or no longer visible
		}
		out = append(out, Conversation{ID: r.ID, MatchID: r.MatchID, User: card, LastMessage: r.Last, UnreadCount: r.UnreadCount, UpdatedAt: r.UpdatedAt})
	}
	return out, nil
}

func (s *Service) UnreadTotal(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadTotal(ctx, userID)
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID, beforeID string, limit int) ([]Message, error) {
	a, err := s.repo.Access(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListMessages(ctx, conversationID, a.ClearedAt, beforeID, limit)
}

func (s *Service) Send(ctx context.Context, userID, conversationID, content string) (Message, error) {
	text, err := normalizeContent(content)
	if err != nil {
		return Message{}, err
	}
	a, err := s.repo.Access(ctx, userID, conversationID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.CreateMessage(ctx, conversationID, userID, text)
	if err != nil {
		return Message{}, err
	}
	event := realtime.Event{Type: "message.new", Data: msg}
	s.pub.Publish(a.OtherUserID, event)
	s.pub.Publish(userID, event) // keep the sender's other devices in sync
	return msg, nil
}

// MarkRead flags the counterpart's messages as read and tells them live.
func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (int, error) {
	a, err := s.repo.Access(ctx, userID, conversationID)
	if err != nil {
		return 0, err
	}
	count, at, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return 0, err
	}
	if count > 0 {
		s.pub.Publish(a.OtherUserID, realtime.Event{Type: "messages.read", Data: map[string]any{
			"conversationId": conversationID, "readerId": userID, "readAt": at,
		}})
	}
	return count, nil
}

func (s *Service) Clear(ctx context.Context, userID, conversationID string) error {
	if _, err := s.repo.Access(ctx, userID, conversationID); err != nil {
		return err
	}
	return s.repo.Clear(ctx, userID, conversationID)
}

func normalizeContent(content string) (string, error) {
	var b strings.Builder
	for _, r := range strings.TrimSpace(content) {
		if unicode.IsControl(r) && r != '\n' {
			continue
		}
		b.WriteRune(r)
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(text) > MaxMessageRunes {
		return "", ErrMessageTooLong
	}
	return text, nil
}
