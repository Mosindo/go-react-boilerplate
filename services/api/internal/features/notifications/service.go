package notifications

import (
	"context"
	"errors"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var ErrNotificationNotFound = apperr.NotFound("notification_not_found", "notification not found")

const (
	DefaultLimit = 20
	MaxLimit     = 50
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

// Notify stores an in-app notification and pushes it over the realtime
// channel. Notifications are only created by the server, never by clients.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]any) error {
	n, err := s.repo.Create(ctx, userID, kind, title, body, data)
	if err != nil {
		return err
	}
	s.publisher.Publish(userID, realtime.Event{Type: "notification.created", Data: n})
	return nil
}

func (s *Service) List(ctx context.Context, userID, beforeID string, limit int) (ListResponse, error) {
	items, err := s.repo.List(ctx, userID, beforeID, limit+1)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{Notifications: items}
	if len(items) > limit {
		resp.Notifications = items[:limit]
		cursor := items[limit-1].ID
		resp.NextCursor = &cursor
	}
	if resp.UnreadCount, err = s.repo.UnreadCount(ctx, userID); err != nil {
		return ListResponse{}, err
	}
	return resp, nil
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) error {
	if err := s.repo.MarkRead(ctx, userID, notificationID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrNotificationNotFound
		}
		return err
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
