package notifications

import (
	"context"
	"log"

	"example.com/api/internal/platform/realtime"
)

const (
	defaultLimit = 20
	maxLimit     = 50
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) ([]Notification, int, error) {
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}
	if offset < 0 {
		offset = 0
	}
	items, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return items, unread, nil
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// Notify stores an in-app notification and pushes it in real time. Failures
// are logged, never returned: a notification must not break the action that
// triggered it (a like, a message).
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string, coalesce bool) {
	n, created, err := s.repo.Create(ctx, Notification{UserID: userID, Type: kind, Title: title, Body: body, Data: data}, coalesce)
	if err != nil {
		log.Printf(`{"event":"notification_create_error","type":%q,"error":%q}`, kind, err.Error())
		return
	}
	if created {
		s.publisher.Publish(userID, realtime.Event{Type: "notification.created", Data: n})
	}
}
