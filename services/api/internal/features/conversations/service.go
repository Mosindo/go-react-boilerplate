package conversations

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

var ErrInvalidMessage = errors.New("message must contain 1 to 2000 characters")

// Notifier stores an in-app notification for a user.
type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) error
}

// Presence tells whether a user currently has a live connection.
type Presence interface {
	Connected(userID string) bool
}

// Names resolves display names for notification texts.
type Names interface {
	FirstName(ctx context.Context, userID string) (string, error)
}

type Service struct {
	repo     Repository
	events   realtime.Publisher
	notifier Notifier
	presence Presence
	names    Names
}

func NewService(repo Repository, events realtime.Publisher, notifier Notifier, presence Presence, names Names) *Service {
	return &Service{repo: repo, events: events, notifier: notifier, presence: presence, names: names}
}

func normalizeLimit(limit int) int {
	switch {
	case limit <= 0:
		return DefaultPageSize
	case limit > MaxPageSize:
		return MaxPageSize
	}
	return limit
}

func (s *Service) List(ctx context.Context, userID string, before *time.Time, limit int) ([]Conversation, error) {
	return s.repo.List(ctx, userID, before, normalizeLimit(limit))
}

// Messages returns the newest-first page of a conversation the caller belongs to.
func (s *Service) Messages(ctx context.Context, userID, matchID string, before *time.Time, limit int) ([]Message, error) {
	access, err := s.repo.Access(ctx, userID, matchID)
	if err != nil {
		return nil, err
	}
	return s.repo.Messages(ctx, matchID, access.HiddenAt, before, normalizeLimit(limit))
}

func cleanBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" || utf8.RuneCountInString(body) > MaxMessageRunes {
		return "", ErrInvalidMessage
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", ErrInvalidMessage
		}
	}
	return body, nil
}

func (s *Service) Send(ctx context.Context, userID, matchID, body string) (Message, error) {
	body, err := cleanBody(body)
	if err != nil {
		return Message{}, err
	}
	access, err := s.repo.Access(ctx, userID, matchID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.Insert(ctx, userID, matchID, body)
	if err != nil {
		return Message{}, err
	}

	event := realtime.Event{Type: "message.new", Data: msg}
	s.events.Publish(userID, event) // other devices of the sender
	s.events.Publish(access.OtherID, event)

	if !s.presence.Connected(access.OtherID) {
		name := "Someone"
		if n, err := s.names.FirstName(ctx, userID); err == nil && n != "" {
			name = n
		}
		_ = s.notifier.Notify(ctx, access.OtherID, "message", "New message from "+name, "You have unread messages.",
			map[string]string{"matchId": matchID, "userId": userID, "name": name})
	}
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, matchID string) error {
	access, err := s.repo.Access(ctx, userID, matchID)
	if err != nil {
		return err
	}
	n, err := s.repo.MarkRead(ctx, userID, matchID)
	if err != nil {
		return err
	}
	if n > 0 {
		s.events.Publish(access.OtherID, realtime.Event{Type: "message.read", Data: map[string]string{"matchId": matchID, "readerId": userID}})
	}
	return nil
}

func (s *Service) Hide(ctx context.Context, userID, matchID string) error {
	return s.repo.Hide(ctx, userID, matchID)
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	access, err := s.repo.Access(ctx, userID, matchID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if err := s.repo.Unmatch(ctx, userID, matchID); err != nil {
		return err
	}
	if access.OtherID != "" {
		s.events.Publish(access.OtherID, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	}
	return nil
}
