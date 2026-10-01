package notifications

import (
	"context"

	"example.com/api/internal/platform/realtime"
)

type Publisher interface {
	Publish(userID string, event realtime.Event)
}

type Service struct {
	repo Repository
	pub  Publisher
}

func NewService(repo Repository, pub Publisher) *Service { return &Service{repo: repo, pub: pub} }

const (
	defaultLimit = 20
	maxLimit     = 100
)

func (s *Service) List(ctx context.Context, userID string, limit, offset int) ([]Notification, int, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	list, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	return list, unread, err
}

// Notify stores a notification and pushes it live. Identical unread
// notifications (same type and data) are collapsed, so a chatty conversation
// produces one entry rather than one per message.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) error {
	n, created, err := s.repo.CreateUnique(ctx, userID, kind, title, body, data)
	if err != nil {
		return err
	}
	if created && s.pub != nil {
		s.pub.Publish(userID, realtime.Event{Type: "notification", Data: n})
	}
	return nil
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) (int, error) {
	return s.repo.MarkAllRead(ctx, userID)
}
