package notifications

import (
	"context"
	"log"

	"example.com/api/internal/platform/realtime"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type Service struct {
	repo Repository
	hub  *realtime.Hub
}

func NewService(repo Repository, hub *realtime.Hub) *Service {
	return &Service{repo: repo, hub: hub}
}

// Notify stores a notification and pushes it to the user's live connections.
// It is best-effort: callers (matching, chat) must not fail because a
// notification could not be written, so errors are logged, not returned.
func (s *Service) Notify(ctx context.Context, n New) {
	created, ok, err := s.repo.Create(ctx, n)
	if err != nil {
		log.Printf(`{"event":"notify_failed","type":%q,"error":%q}`, n.Type, err.Error())
		return
	}
	if ok && s.hub != nil {
		s.hub.Publish(n.UserID, realtime.Event{Type: "notification.new", Data: ToResponse(created)})
	}
}

func (s *Service) List(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]Notification, int, error) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	items, err := s.repo.List(ctx, userID, unreadOnly, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.repo.CountUnread(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return items, unread, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) (Notification, error) {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
