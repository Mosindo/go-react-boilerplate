package notifications

import (
	"context"
	"time"

	"example.com/api/internal/platform/realtime"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type Service struct {
	repo   Repository
	events realtime.Publisher
}

func NewService(repo Repository, events realtime.Publisher) *Service {
	return &Service{repo: repo, events: events}
}

func (s *Service) List(ctx context.Context, userID string, before *time.Time, limit int) ([]Notification, int, error) {
	switch {
	case limit <= 0:
		limit = defaultLimit
	case limit > maxLimit:
		limit = maxLimit
	}
	items, err := s.repo.List(ctx, userID, before, limit)
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return items, unread, nil
}

// Notify is the internal entry point other features use; there is deliberately no HTTP
// endpoint to create notifications.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) error {
	n, err := s.repo.Create(ctx, userID, kind, title, body, data)
	if err != nil || n == nil {
		return err
	}
	s.events.Publish(userID, realtime.Event{Type: "notification.new", Data: n})
	return nil
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) error {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
