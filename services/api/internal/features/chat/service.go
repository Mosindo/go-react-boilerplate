package chat

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalidBody   = errors.New("message must be 1-2000 characters")
	ErrInvalidCursor = errors.New("invalid cursor")
)

type Service struct {
	repo   Repository
	events Events
}

func NewService(repo Repository, events Events) *Service {
	return &Service{repo: repo, events: events}
}

func (s *Service) List(ctx context.Context, userID string, limit int, cursor string) (ConversationPage, error) {
	var at *time.Time
	var id string
	if cursor != "" {
		t, cid, ok := decodeCursor(cursor)
		if !ok {
			return ConversationPage{}, ErrInvalidCursor
		}
		at, id = &t, cid
	}
	items, err := s.repo.List(ctx, userID, limit+1, at, id)
	if err != nil {
		return ConversationPage{}, err
	}
	page := ConversationPage{Conversations: items}
	if len(items) > limit {
		page.Conversations = items[:limit]
		last := page.Conversations[limit-1]
		page.NextCursor = encodeCursor(last.Activity, last.ID)
	}
	return page, nil
}

func (s *Service) Messages(ctx context.Context, userID, convID string, beforeID int64, limit int) ([]Message, error) {
	return s.repo.Messages(ctx, userID, convID, beforeID, limit)
}

func (s *Service) Send(ctx context.Context, userID, convID, body string) (Message, error) {
	body, err := CleanBody(body)
	if err != nil {
		return Message{}, err
	}
	msg, name, recipient, err := s.repo.Send(ctx, userID, convID, body)
	if err != nil {
		return Message{}, err
	}
	if s.events != nil {
		s.events.MessageCreated(msg, name, recipient)
	}
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, convID string) error {
	other, upTo, err := s.repo.MarkRead(ctx, userID, convID)
	if err != nil {
		return err
	}
	if s.events != nil {
		s.events.MessagesRead(convID, userID, other, upTo)
	}
	return nil
}

func (s *Service) Hide(ctx context.Context, userID, convID string) error {
	return s.repo.Hide(ctx, userID, convID)
}

// CleanBody trims a message and rejects empty, oversized or control-char input.
func CleanBody(body string) (string, error) {
	body = strings.TrimSpace(body)
	n := utf8.RuneCountInString(body)
	if n < 1 || n > MaxBodyRunes || !utf8.ValidString(body) {
		return "", ErrInvalidBody
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", ErrInvalidBody
		}
	}
	return body, nil
}

func encodeCursor(t time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(t.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeCursor(c string) (time.Time, string, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, "", false
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", false
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", false
	}
	return t, parts[1], true
}
