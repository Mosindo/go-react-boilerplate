package notifications

import (
	"context"
	"time"

	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo *Repository
	pub  Publisher
}

func NewService(repo *Repository, pub Publisher) *Service { return &Service{repo: repo, pub: pub} }

// Notify records a notification and tells connected clients to update their badge.
func (s *Service) Notify(ctx context.Context, userID, kind, actorID, refID string) error {
	id, err := s.repo.Upsert(ctx, userID, kind, actorID, refID)
	if err != nil {
		return err
	}
	s.pub.Publish(userID, realtime.Event{Type: "notification.new", Data: map[string]string{"id": id, "type": kind, "refId": refID}})
	return nil
}

// MarkRefRead marks unread notifications of a kind about ref (e.g. a conversation) as read.
func (s *Service) MarkRefRead(ctx context.Context, userID, kind, refID string) error {
	return s.repo.MarkRefRead(ctx, userID, kind, refID)
}

func (s *Service) List(ctx context.Context, userID string, limit int, before *time.Time) ([]Notification, error) {
	return s.repo.List(ctx, userID, limit, before)
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) (bool, error) {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
