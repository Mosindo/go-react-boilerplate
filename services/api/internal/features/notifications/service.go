package notifications

import (
	"context"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/pagination"
	"github.com/google/uuid"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID string, limit int, cursorValue string) ([]Notification, string, error) {
	cursor, err := pagination.Decode(cursorValue)
	if err != nil {
		return nil, "", err
	}
	items, err := s.repo.List(ctx, userID, limit+1, cursor)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		next = pagination.Encode(last.CreatedAt, last.ID)
	}
	return items, next, nil
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return platformerrors.NotFound("Notification not found")
	}
	found, err := s.repo.MarkRead(ctx, userID, id)
	if err != nil {
		return err
	}
	if !found {
		return platformerrors.NotFound("Notification not found")
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
