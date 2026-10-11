package notifications

import (
	"context"
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
)

var ErrNotFound = platformerrors.New(http.StatusNotFound, "not_found", "notification not found")
var ErrBadCursor = platformerrors.New(http.StatusBadRequest, "invalid_request", "invalid cursor")

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

type Page struct {
	Notifications []Notification `json:"notifications"`
	UnreadCount   int            `json:"unreadCount"`
	NextCursor    *string        `json:"nextCursor"`
}

func decodeCursor(raw string) (*Cursor, error) {
	t, id, ok, err := httpx.DecodeCursor(raw)
	if err != nil {
		return nil, ErrBadCursor
	}
	if !ok {
		return nil, nil
	}
	return &Cursor{CreatedAt: t, ID: id}, nil
}

func (s *Service) List(ctx context.Context, userID, cursor string, limit int) (Page, error) {
	cur, err := decodeCursor(cursor)
	if err != nil {
		return Page{}, err
	}
	items, err := s.repo.List(ctx, userID, cur, limit+1)
	if err != nil {
		return Page{}, err
	}
	page := Page{}
	if len(items) > limit {
		items = items[:limit]
		next := httpx.EncodeCursor(items[len(items)-1].CreatedAt, items[len(items)-1].ID)
		page.NextCursor = &next
	}
	page.Notifications = items
	page.UnreadCount, err = s.repo.UnreadCount(ctx, userID)
	return page, err
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
