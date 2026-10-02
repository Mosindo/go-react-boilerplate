package notifications

import (
	"context"
	"encoding/base64"
	"errors"
	"strconv"
	"time"

	"example.com/api/internal/features/profiles"
)

var (
	ErrNotFound  = errors.New("notification not found")
	ErrBadCursor = errors.New("invalid cursor")
)

type CardSource interface {
	Cards(ctx context.Context, viewerID string, ids []string) ([]profiles.Card, error)
}

type Service struct {
	repo  Repository
	cards CardSource
}

func NewService(repo Repository, cards CardSource) *Service {
	return &Service{repo: repo, cards: cards}
}

func (s *Service) List(ctx context.Context, userID, cursor string, limit int) (Page, error) {
	var before *time.Time
	if cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return Page{}, ErrBadCursor
		}
		micro, err := strconv.ParseInt(string(raw), 10, 64)
		if err != nil {
			return Page{}, ErrBadCursor
		}
		t := time.UnixMicro(micro).UTC()
		before = &t
	}
	rows, err := s.repo.List(ctx, userID, before, limit+1)
	if err != nil {
		return Page{}, err
	}
	page := Page{Notifications: []Item{}}
	if len(rows) > limit {
		rows = rows[:limit]
		page.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(rows[len(rows)-1].CreatedAt.UnixMicro(), 10)))
	}
	ids := make([]string, 0, len(rows))
	seen := map[string]bool{}
	for _, r := range rows {
		if !seen[r.ActorID] {
			seen[r.ActorID] = true
			ids = append(ids, r.ActorID)
		}
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return Page{}, err
	}
	byID := make(map[string]profiles.Card, len(cards))
	for _, c := range cards {
		byID[c.ID] = c
	}
	for _, r := range rows {
		item := Item{ID: r.ID, Type: r.Type, MatchID: r.MatchID, CreatedAt: r.CreatedAt, ReadAt: r.ReadAt}
		if c, ok := byID[r.ActorID]; ok {
			item.Actor = &c
		}
		page.Notifications = append(page.Notifications, item)
	}
	sum, err := s.repo.Summary(ctx, userID)
	if err != nil {
		return Page{}, err
	}
	page.Unread = sum.UnreadNotifications
	return page, nil
}

func (s *Service) Summary(ctx context.Context, userID string) (Summary, error) {
	return s.repo.Summary(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if err := s.repo.MarkRead(ctx, userID, id); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
