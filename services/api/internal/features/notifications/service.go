package notifications

import (
	"context"
	"time"

	"example.com/api/internal/features/profiles"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID string, before *time.Time, limit int) (ListResponse, error) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	list, err := s.repo.List(ctx, userID, before, limit)
	if err != nil {
		return ListResponse{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{Notifications: list, UnreadCount: unread}
	if len(list) == limit {
		last := list[len(list)-1].CreatedAt
		resp.NextCursor = &last
	}
	return resp, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if !profiles.IsUUID(id) {
		return ErrNotFound
	}
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
