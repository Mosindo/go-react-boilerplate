package notifications

import (
	"context"
	"errors"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID string, before *time.Time, limit int) (ListResponse, error) {
	// Fetch one extra row to know whether another page exists.
	items, err := s.repo.List(ctx, userID, before, limit+1)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{}
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1].CreatedAt
		resp.NextBefore = &last
	}
	resp.Notifications = items
	if resp.Notifications == nil {
		resp.Notifications = []Notification{}
	}
	resp.UnreadCount, err = s.repo.UnreadCount(ctx, userID)
	return resp, err
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if err := s.repo.MarkRead(ctx, userID, id); err != nil {
		if errors.Is(err, ErrNotFound) {
			return platformerrors.NotFound("notification not found")
		}
		return err
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
