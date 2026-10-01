package chat

import (
	"context"
	"log"
	"time"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

type Cards interface {
	Cards(ctx context.Context, viewerID string, ids []string) (map[string]profiles.Card, error)
}

type Notifier interface {
	Notify(ctx context.Context, userID, kind, actorID, refID string) error
	MarkRefRead(ctx context.Context, userID, kind, refID string) error
}

type Publisher interface {
	Publish(userID string, ev realtime.Event)
}

type Service struct {
	repo     *Repository
	cards    Cards
	notifier Notifier
	pub      Publisher
}

func NewService(repo *Repository, cards Cards, notifier Notifier, pub Publisher) *Service {
	return &Service{repo: repo, cards: cards, notifier: notifier, pub: pub}
}

func (s *Service) Conversations(ctx context.Context, userID string, limit, offset int) ([]Conversation, error) {
	rows, err := s.repo.Conversations(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, c := range rows {
		ids = append(ids, c.OtherID)
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Conversation, 0, len(rows))
	for _, c := range rows {
		card, ok := cards[c.OtherID]
		if !ok {
			continue
		}
		out = append(out, Conversation{ID: c.ID, MatchID: c.MatchID, User: card, LastMessage: c.Last, UnreadCount: c.Unread, LastMessageAt: c.LastMessageAt})
	}
	return out, nil
}

func (s *Service) Messages(ctx context.Context, userID, conversationID string, limit int, before *time.Time) ([]Message, error) {
	if _, err := s.repo.OtherParticipant(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	return s.repo.Messages(ctx, userID, conversationID, limit, before)
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	body = profiles.CleanText(body)
	if body == "" {
		return Message{}, ErrEmptyMessage
	}
	if utf8.RuneCountInString(body) > MaxMessageRunes {
		return Message{}, ErrTooLong
	}
	msg, recipient, err := s.repo.Send(ctx, userID, conversationID, body)
	if err != nil {
		return Message{}, err
	}
	s.pub.Publish(recipient, realtime.Event{Type: "message.new", Data: msg})
	if err := s.notifier.Notify(ctx, recipient, "message", userID, conversationID); err != nil {
		log.Printf("chat: notify message: %v", err)
	}
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (int64, error) {
	other, err := s.repo.OtherParticipant(ctx, userID, conversationID)
	if err != nil {
		return 0, err
	}
	n, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		return 0, err
	}
	if err := s.notifier.MarkRefRead(ctx, userID, "message", conversationID); err != nil {
		log.Printf("chat: mark notification read: %v", err)
	}
	if n > 0 {
		s.pub.Publish(other, realtime.Event{Type: "message.read", Data: map[string]string{"conversationId": conversationID, "readerId": userID}})
	}
	return n, nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	return s.repo.Hide(ctx, userID, conversationID)
}
