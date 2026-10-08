package notifications

import (
	"context"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/pagination"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID string, limit int, cursor string, unreadOnly bool) (ListResponse, error) {
	cur, err := pagination.Decode(cursor)
	if err != nil {
		return ListResponse{}, err
	}
	var afterAt *time.Time
	afterID := ""
	if cur != nil {
		afterAt, afterID = &cur.At, cur.ID
	}
	items, err := s.repo.List(ctx, userID, limit+1, unreadOnly, afterAt, afterID)
	if err != nil {
		return ListResponse{}, err
	}
	res := ListResponse{}
	if len(items) > limit {
		last := items[limit-1]
		res.NextCursor = pagination.Cursor{At: last.CreatedAt, ID: last.ID}.Encode()
		items = items[:limit]
	}
	res.Notifications = items
	return res, nil
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	ok, err := s.repo.MarkRead(ctx, userID, id)
	if err != nil {
		return err
	}
	if !ok {
		return platformerrors.NotFound("not_found", "notification not found")
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
