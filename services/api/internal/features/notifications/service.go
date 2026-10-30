package notifications

import (
	"context"
	"time"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID string, before time.Time, limit int) (ListResponse, error) {
	items, err := s.repo.List(ctx, userID, before, limit+1)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{}
	if len(items) > limit {
		items = items[:limit]
		next := items[len(items)-1].CreatedAt
		resp.NextBefore = &next
	}
	resp.Notifications = items
	resp.UnreadCount, err = s.repo.UnreadCount(ctx, userID)
	return resp, err
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) error {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
