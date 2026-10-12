package alerts

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
)

const (
	DefaultLimit = 30
	MaxLimit     = 50
)

var ErrNotFound = errors.New("notification not found")

type Cards interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) ([]profiles.Card, error)
}

type Service struct {
	repo  Repository
	cards Cards
}

func NewService(repo Repository, cards Cards) *Service { return &Service{repo: repo, cards: cards} }

func (s *Service) List(ctx context.Context, userID, beforeID string, limit int) (Page, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	stored, err := s.repo.List(ctx, userID, beforeID, limit+1)
	if err != nil {
		return Page{}, err
	}
	page := Page{}
	if len(stored) > limit {
		stored = stored[:limit]
		page.NextBefore = stored[len(stored)-1].ID
	}
	ids := []string{}
	for _, n := range stored {
		if n.ActorID != nil {
			ids = append(ids, *n.ActorID)
		}
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return Page{}, err
	}
	actors := map[string]*Actor{}
	for _, c := range cards {
		a := &Actor{UserID: c.UserID, FirstName: c.FirstName}
		if len(c.Photos) > 0 {
			a.ThumbURL = c.Photos[0].ThumbURL
		}
		actors[c.UserID] = a
	}
	page.Items = make([]Notification, 0, len(stored))
	for _, n := range stored {
		item := Notification{ID: n.ID, Kind: n.Kind, MatchID: n.MatchID, ReadAt: n.ReadAt, CreatedAt: n.CreatedAt}
		if n.ActorID != nil {
			item.Actor = actors[*n.ActorID]
		}
		page.Items = append(page.Items, item)
	}
	if page.UnreadCount, err = s.repo.UnreadCount(ctx, userID); err != nil {
		return Page{}, err
	}
	return page, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	ok, err := s.repo.MarkRead(ctx, userID, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
