package notifications

import (
	"context"
	"log"

	"example.com/api/internal/platform/realtime"
)

const (
	defaultLimit = 20
	MaxLimit     = 50
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) List(ctx context.Context, userID, beforeID string, limit int) (ListResponse, error) {
	if limit <= 0 || limit > MaxLimit {
		limit = defaultLimit
	}
	items, err := s.repo.List(ctx, userID, beforeID, limit+1)
	if err != nil {
		return ListResponse{}, err
	}
	var next *string
	if len(items) > limit {
		items = items[:limit]
		cursor := items[len(items)-1].ID
		next = &cursor
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Notifications: items, NextCursor: next, UnreadCount: unread}, nil
}

// Notify persists an in-app notification and pushes it in real time. It is called by
// other features after their own transaction committed; failures are logged, never
// propagated, so a notification problem cannot undo a match or a message.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) {
	n, err := s.repo.Create(ctx, userID, kind, title, body, data)
	if err != nil {
		log.Printf(`{"event":"notification_create_failed","user_id":%q,"type":%q,"error":%q}`, userID, kind, err.Error())
		return
	}
	if s.publisher != nil {
		s.publisher.Publish(userID, "notification.created", n)
	}
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

// NotifyCollapsed is Notify with de-duplication on collapseKey while unread.
func (s *Service) NotifyCollapsed(ctx context.Context, userID, kind, title, body string, data map[string]string, collapseKey string) {
	n, created, err := s.repo.CreateCollapsed(ctx, userID, kind, title, body, data, collapseKey)
	if err != nil {
		log.Printf(`{"event":"notification_create_failed","user_id":%q,"type":%q,"error":%q}`, userID, kind, err.Error())
		return
	}
	if created && s.publisher != nil {
		s.publisher.Publish(userID, "notification.created", n)
	}
}
