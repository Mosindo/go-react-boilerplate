package notifications

import (
	"context"
	"errors"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var ErrNotificationNotFound = platformerrors.NotFound("notification_not_found", "notification not found")

// Publisher pushes a live event to a user; *realtime.Hub implements it.
type Publisher interface {
	Publish(userID string, event realtime.Event)
}

type Service struct {
	repo      Repository
	publisher Publisher
}

func NewService(repo Repository, publisher Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (ListResponse, error) {
	list, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return ListResponse{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Notifications: list, UnreadCount: unread}, nil
}

// Notify stores a notification and pushes it to the user's open connections.
// Other modules call it; failures are returned so callers can decide to log and continue.
func (s *Service) Notify(ctx context.Context, userID string, n NewNotification) error {
	created, err := s.repo.Create(ctx, userID, n)
	if err != nil {
		return err
	}
	s.publisher.Publish(userID, realtime.Event{Type: "notification.new", Data: created})
	return nil
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	n, err := s.repo.MarkRead(ctx, userID, notificationID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return Notification{}, ErrNotificationNotFound
	}
	return n, err
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) MarkConversationRead(ctx context.Context, userID, conversationID string) error {
	return s.repo.MarkConversationRead(ctx, userID, conversationID)
}
