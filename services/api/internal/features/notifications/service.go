package notifications

import (
	"context"
	"time"

	"example.com/api/internal/platform/httpx"
)

const (
	DefaultLimit = 20
	MaxLimit     = 50
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID, rawCursor string, limit int) (ListResponse, error) {
	var cursor *Cursor
	if rawCursor != "" {
		var c Cursor
		if err := httpx.DecodeCursor(rawCursor, &c); err != nil {
			return ListResponse{}, err
		}
		if !httpx.IsUUID(c.ID) || c.CreatedAt.IsZero() {
			return ListResponse{}, httpx.BadRequest("invalid cursor")
		}
		cursor = &c
	}
	items, err := s.repo.List(ctx, userID, cursor, limit+1)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{Items: items}
	if len(items) > limit {
		resp.Items = items[:limit]
		last := resp.Items[limit-1]
		next := httpx.EncodeCursor(Cursor{CreatedAt: last.CreatedAt.Truncate(time.Microsecond), ID: last.ID})
		resp.NextCursor = &next
	}
	if resp.Items == nil {
		resp.Items = []Notification{}
	}
	resp.UnreadCount, err = s.repo.UnreadCount(ctx, userID)
	return resp, err
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if !httpx.IsUUID(id) {
		return httpx.NotFound("notification not found")
	}
	found, err := s.repo.MarkRead(ctx, userID, id)
	if err != nil {
		return err
	}
	if !found {
		return httpx.NotFound("notification not found")
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
