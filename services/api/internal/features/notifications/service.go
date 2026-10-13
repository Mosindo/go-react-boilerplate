package notifications

import (
	"context"
	"time"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID string, before *time.Time, limit int) (ListResponse, error) {
	if limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	items, err := s.repo.List(ctx, userID, before, limit)
	if err != nil {
		return ListResponse{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{Notifications: items, UnreadCount: unread}
	if len(items) == limit {
		last := items[len(items)-1].CreatedAt
		resp.NextBefore = &last
	}
	return resp, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
