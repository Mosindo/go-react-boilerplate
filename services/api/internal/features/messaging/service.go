package messaging

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

const (
	MaxBodyRunes = 2000
	DefaultLimit = 30
	MaxLimit     = 100
)

var (
	ErrNotFound   = errors.New("conversation not found")
	ErrValidation = errors.New("message must be between 1 and 2000 characters")
)

type Service struct {
	repo   Repository
	events realtime.Publisher
}

func NewService(repo Repository, events realtime.Publisher) *Service {
	return &Service{repo: repo, events: events}
}

func (s *Service) member(ctx context.Context, matchID, userID string) (Membership, error) {
	m, err := s.repo.Membership(ctx, matchID, userID)
	if errors.Is(err, ErrRepositoryNotMember) {
		return Membership{}, ErrNotFound
	}
	return m, err
}

// CleanBody trims and validates a message body: valid UTF-8, no control
// characters other than newline/tab, 1..2000 runes.
func CleanBody(raw string) (string, error) {
	body := strings.TrimSpace(raw)
	if body == "" || !utf8.ValidString(body) || utf8.RuneCountInString(body) > MaxBodyRunes {
		return "", ErrValidation
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", ErrValidation
		}
	}
	return body, nil
}

func (s *Service) List(ctx context.Context, userID, matchID, beforeID string, limit int) (Page, error) {
	m, err := s.member(ctx, matchID, userID)
	if err != nil {
		return Page{}, err
	}
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	items, err := s.repo.List(ctx, matchID, m.ClearedAt, beforeID, limit+1)
	if err != nil {
		return Page{}, err
	}
	page := Page{OtherLastReadAt: m.OtherLastReadAt}
	if len(items) > limit {
		items = items[:limit]
		page.NextBefore = items[len(items)-1].ID
	}
	page.Items = items
	return page, nil
}

func (s *Service) Send(ctx context.Context, userID, matchID, rawBody string) (Message, error) {
	body, err := CleanBody(rawBody)
	if err != nil {
		return Message{}, err
	}
	m, err := s.member(ctx, matchID, userID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.Insert(ctx, matchID, userID, m.OtherUserID, body)
	if err != nil {
		return Message{}, err
	}
	for _, uid := range []string{m.OtherUserID, userID} {
		s.events.Publish(uid, realtime.Event{Type: "message.new", Data: msg})
	}
	s.events.Publish(m.OtherUserID, realtime.Event{Type: "notification.new"})
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, matchID string) error {
	m, err := s.member(ctx, matchID, userID)
	if err != nil {
		return err
	}
	readAt, err := s.repo.MarkRead(ctx, matchID, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotMember) {
			return ErrNotFound
		}
		return err
	}
	s.events.Publish(m.OtherUserID, realtime.Event{Type: "message.read", Data: map[string]any{"matchId": matchID, "readAt": readAt}})
	return nil
}

func (s *Service) Clear(ctx context.Context, userID, matchID string) error {
	if _, err := s.member(ctx, matchID, userID); err != nil {
		return err
	}
	if err := s.repo.Clear(ctx, matchID, userID); err != nil {
		if errors.Is(err, ErrRepositoryNotMember) {
			return ErrNotFound
		}
		return err
	}
	return nil
}
