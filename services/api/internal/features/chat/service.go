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
	ErrMessageTooLong = errors.New("message is too long (max 2000 characters)")
	ErrNotFound       = errors.New("conversation not found")
)

type Service struct {
	repo   Repository
	events realtime.Publisher
}

func NewService(repo Repository, events realtime.Publisher) *Service {
	return &Service{repo: repo, events: events}
}

// CleanBody trims the text and rejects empty, oversized or control-character payloads.
func CleanBody(raw string) (string, error) {
	body := strings.TrimSpace(raw)
	if body == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(body) > MaxBodyLen {
		return "", ErrMessageTooLong
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", ErrEmptyMessage
		}
	}
	return body, nil
}

func (s *Service) Send(ctx context.Context, matchID, senderID, raw string) (Message, error) {
	body, err := CleanBody(raw)
	if err != nil {
		return Message{}, err
	}
	m, recipient, err := s.repo.Insert(ctx, matchID, senderID, body)
	if err != nil {
		if errors.Is(err, ErrRepoNotParticipant) {
			return Message{}, ErrNotFound
		}
		return Message{}, err
	}
	ev := realtime.Event{Type: "message", Payload: m}
	s.events.Publish(recipient, ev)
	s.events.Publish(senderID, ev) // other devices of the sender
	return m, nil
}

// List returns up to limit messages older than beforeID (0 = newest), oldest first.
func (s *Service) List(ctx context.Context, matchID, userID string, beforeID int64, limit int) ([]Message, error) {
	msgs, err := s.repo.List(ctx, matchID, userID, beforeID, limit)
	if err != nil {
		if errors.Is(err, ErrRepoNotParticipant) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, nil
}

func (s *Service) MarkRead(ctx context.Context, matchID, userID string) (int, error) {
	n, other, err := s.repo.MarkRead(ctx, matchID, userID)
	if err != nil {
		if errors.Is(err, ErrRepoNotParticipant) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	if n > 0 {
		s.events.Publish(other, realtime.Event{Type: "read", Payload: map[string]string{"matchId": matchID}})
	}
	return n, nil
}

func (s *Service) Clear(ctx context.Context, matchID, userID string) error {
	if err := s.repo.Clear(ctx, matchID, userID); err != nil {
		if errors.Is(err, ErrRepoNotParticipant) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
