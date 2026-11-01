package notifications

import (
	"context"

	"example.com/api/internal/platform/httpx"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, userID string, limit int, cur *httpx.Cursor) ([]Notification, int, error) {
	items, err := s.repo.List(ctx, userID, limit, cur)
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	return items, unread, err
}

func (s *Service) MarkRead(ctx context.Context, userID string, ids []string) error {
	return s.repo.MarkRead(ctx, userID, ids)
}
