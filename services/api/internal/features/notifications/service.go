package notifications

import (
	"context"
	"errors"

	platformerrors "example.com/api/internal/platform/errors"
)

type Publisher interface {
	Publish(userID, eventType string, data any)
}

type Service struct {
	repo   *Repository
	events Publisher
}

func NewService(repo *Repository, events Publisher) *Service {
	return &Service{repo: repo, events: events}
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (ListResponse, error) {
	items, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return ListResponse{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResponse{}, err
	}
	if items == nil {
		items = []Notification{}
	}
	return ListResponse{Items: items, UnreadCount: unread}, nil
}

// Notify stores a notification and pushes it to the user's live connections.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) error {
	n, err := s.repo.Create(ctx, userID, kind, title, body, data)
	if err != nil {
		return err
	}
	s.events.Publish(userID, "notification", n)
	return nil
}

func (s *Service) NotifyMessage(ctx context.Context, userID, conversationID, fromUserID, title, preview string) error {
	n, err := s.repo.UpsertMessage(ctx, userID, conversationID, title, preview,
		map[string]string{"conversationId": conversationID, "userId": fromUserID})
	if err != nil {
		return err
	}
	s.events.Publish(userID, "notification", n)
	return nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) (Notification, error) {
	n, err := s.repo.MarkRead(ctx, userID, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return Notification{}, platformerrors.NotFound("notification not found")
		}
		return Notification{}, err
	}
	return n, nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
