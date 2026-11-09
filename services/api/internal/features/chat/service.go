package chat

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	rt   realtime.Publisher
}

func NewService(repo Repository, rt realtime.Publisher) *Service { return &Service{repo: repo, rt: rt} }

// NormalizeBody trims and validates message text.
func NormalizeBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(body) > MaxMessageLength {
		return "", ErrTooLong
	}
	for _, r := range body {
		if r < 0x20 && r != '\n' && r != '\t' {
			return "", ErrEmptyMessage
		}
	}
	return body, nil
}

func (s *Service) Conversations(ctx context.Context, userID string, before *time.Time, limit int) ([]Conversation, error) {
	return s.repo.ListConversations(ctx, userID, before, limit)
}

func (s *Service) Messages(ctx context.Context, convID, userID, beforeID string, limit int) ([]Message, error) {
	return s.repo.ListMessages(ctx, convID, userID, beforeID, limit)
}

func (s *Service) Send(ctx context.Context, convID, userID, body string) (Message, error) {
	body, err := NormalizeBody(body)
	if err != nil {
		return Message{}, err
	}
	peer, err := s.repo.Peer(ctx, convID, userID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.CreateMessage(ctx, convID, userID, peer, body)
	if err != nil {
		return Message{}, err
	}
	ev := realtime.Event{Type: "message", Data: msg}
	s.rt.Publish(peer, ev)
	s.rt.Publish(userID, ev) // keeps the sender's other devices in sync
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, convID, userID string) error {
	peer, err := s.repo.Peer(ctx, convID, userID)
	if err != nil {
		return err
	}
	n, err := s.repo.MarkRead(ctx, convID, userID)
	if err != nil {
		return err
	}
	if n > 0 {
		s.rt.Publish(peer, realtime.Event{Type: "read", Data: map[string]string{"conversationId": convID}})
	}
	return nil
}

func (s *Service) Hide(ctx context.Context, convID, userID string) error {
	if _, err := s.repo.Peer(ctx, convID, userID); err != nil {
		return err
	}
	return s.repo.Hide(ctx, convID, userID)
}

func (s *Service) UnreadTotal(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadTotal(ctx, userID)
}
