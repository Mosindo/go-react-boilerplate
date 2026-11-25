package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var ErrNotificationNotFound = apperr.New(http.StatusNotFound, "notification_not_found", "notification not found")

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

// Notify stores a notification and pushes it in real time. Notifications are
// only ever produced by the server (matches, system events); clients cannot
// create them.
func (s *Service) Notify(ctx context.Context, userID, typ, title, body string, data map[string]string) {
	raw, err := json.Marshal(data)
	if err != nil {
		raw = []byte("{}")
	}
	n, err := s.repo.Create(ctx, userID, typ, title, body, raw)
	if err != nil {
		log.Printf(`{"event":"notification_create_failed","type":%q,"error":%q}`, typ, err.Error())
		return
	}
	s.pub.Publish(userID, realtime.Event{Type: "notification.created", Data: n})
}

func (s *Service) List(ctx context.Context, userID, beforeID string, limit int) (ListResponse, error) {
	list, err := s.repo.List(ctx, userID, beforeID, limit+1)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{Notifications: list}
	if len(list) > limit {
		resp.Notifications = list[:limit]
		cursor := list[limit-1].ID
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

func (s *Service) MarkRead(ctx context.Context, userID, id string) (Notification, error) {
	n, err := s.repo.MarkRead(ctx, userID, id)
	if errors.Is(err, errRepoNotFound) {
		return Notification{}, ErrNotificationNotFound
	}
	return n, err
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
