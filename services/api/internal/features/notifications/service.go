package notifications

import (
	"context"
	"time"

	"example.com/api/internal/platform/pagination"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, userID string, limit int, cursor string) (ListResponse, error) {
	limit = pagination.Limit(limit, DefaultLimit, MaxLimit)
	var beforeAt *time.Time
	var beforeID string
	if cursor != "" {
		at, id, err := pagination.Decode(cursor)
		if err != nil {
			return ListResponse{}, ErrInvalidCursor
		}
		beforeAt, beforeID = &at, id
	}

	items, err := s.repo.List(ctx, userID, limit+1, beforeAt, beforeID)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{}
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		resp.NextCursor = pagination.Encode(last.CreatedAt, last.ID)
	}
	resp.Notifications = items
	if resp.UnreadCount, err = s.repo.UnreadCount(ctx, userID); err != nil {
		return ListResponse{}, err
	}
	return resp, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
