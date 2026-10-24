package notifications

import (
	"context"
	"log"
	"time"

	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) List(ctx context.Context, userID string, unreadOnly bool, limit, offset int) ([]Notification, error) {
	return s.repo.List(ctx, userID, unreadOnly, limit, offset)
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) (Notification, error) {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// Notify persists a notification and pushes it live. It is best-effort by design:
// a failed notification must never fail the action that triggered it.
func (s *Service) Notify(userID string, n New) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	created, err := s.repo.Create(ctx, userID, n)
	if err != nil {
		log.Printf(`{"event":"notify_failed","user_id":%q,"error":%q}`, userID, err.Error())
		return
	}
	s.pub.Publish(userID, realtime.Event{Type: "notification.new", Data: created})
}
