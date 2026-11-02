package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	perr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

type Notifier interface {
	Create(ctx context.Context, userID, typ string, data map[string]any) error
}

type Service struct {
	repo     *Repository
	hub      *realtime.Hub
	notifier Notifier
	names    NameLookup
}

// NameLookup resolves a display name for notification payloads.
type NameLookup interface {
	FirstName(ctx context.Context, userID string) (string, error)
}

func NewService(repo *Repository, hub *realtime.Hub, notifier Notifier, names NameLookup) *Service {
	return &Service{repo: repo, hub: hub, notifier: notifier, names: names}
}

func translate(err error) error {
	if errors.Is(err, ErrNotFound) {
		return perr.NotFound("Conversation introuvable.")
	}
	return err
}

// CleanBody trims the message and rejects empty, oversized or control-char
// content. Newlines and tabs are kept.
func CleanBody(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", perr.BadRequest("empty_message", "Le message est vide.")
	}
	if utf8.RuneCountInString(s) > MaxMessageRunes {
		return "", perr.BadRequest("message_too_long", "Le message dépasse 2000 caractères.")
	}
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", perr.BadRequest("invalid_message", "Le message contient des caractères invalides.")
		}
	}
	return s, nil
}

func (s *Service) List(ctx context.Context, userID string, limit int, before *time.Time) ([]Conversation, error) {
	return s.repo.List(ctx, userID, limit, before)
}

func (s *Service) Messages(ctx context.Context, matchID, userID string, limit int, before *int64) ([]Message, error) {
	m, err := s.repo.Messages(ctx, matchID, userID, limit, before)
	return m, translate(err)
}

func (s *Service) Send(ctx context.Context, matchID, userID, body string) (Message, error) {
	clean, err := CleanBody(body)
	if err != nil {
		return Message{}, err
	}
	msg, partnerID, err := s.repo.Send(ctx, matchID, userID, clean)
	if err != nil {
		return Message{}, translate(err)
	}
	ev := realtime.Event{Type: "message.new", Data: msg}
	s.hub.Publish(partnerID, ev)
	s.hub.Publish(userID, ev) // sender's other devices
	name, _ := s.names.FirstName(ctx, userID)
	preview := clean
	if utf8.RuneCountInString(preview) > 80 {
		preview = string([]rune(preview)[:80]) + "…"
	}
	_ = s.notifier.Create(ctx, partnerID, "message", map[string]any{
		"matchId": matchID, "userId": userID, "firstName": name, "preview": preview,
	})
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, matchID, userID string) error {
	partnerID, n, err := s.repo.MarkRead(ctx, matchID, userID)
	if err != nil {
		return translate(err)
	}
	if n > 0 {
		s.hub.Publish(partnerID, realtime.Event{Type: "message.read", Data: map[string]any{
			"matchId": matchID, "readerId": userID, "readAt": time.Now().UTC(),
		}})
	}
	return nil
}

func (s *Service) Hide(ctx context.Context, matchID, userID string) error {
	return translate(s.repo.Hide(ctx, matchID, userID))
}

func (s *Service) Unmatch(ctx context.Context, matchID, userID string) error {
	partnerID, err := s.repo.Unmatch(ctx, matchID, userID)
	if err != nil {
		return translate(err)
	}
	ev := realtime.Event{Type: "match.removed", Data: map[string]any{"matchId": matchID}}
	s.hub.Publish(partnerID, ev)
	s.hub.Publish(userID, ev)
	return nil
}
