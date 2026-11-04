package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

// Notifier is the interface other features use to emit notifications.
type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string)
	// NotifyOnce coalesces: no new notification while one is unread for data[key].
	NotifyOnce(ctx context.Context, userID, kind, title, body string, data map[string]string, key string)
}

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

// Notify persists a notification and pushes it in real time. Failures are
// logged and never break the business action that triggered the notification.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) {
	raw, _ := json.Marshal(data)
	n, err := s.repo.Create(ctx, userID, kind, title, body, raw)
	if err != nil {
		log.Printf(`{"event":"notify_error","error":%q}`, err.Error())
		return
	}
	s.pub.Publish(userID, realtime.Event{Type: "notification.new", Data: n})
}

func (s *Service) NotifyOnce(ctx context.Context, userID, kind, title, body string, data map[string]string, key string) {
	raw, _ := json.Marshal(data)
	n, err := s.repo.CreateUnique(ctx, userID, kind, title, body, raw, key, data[key])
	if err != nil {
		log.Printf(`{"event":"notify_error","error":%q}`, err.Error())
		return
	}
	if n != nil {
		s.pub.Publish(userID, realtime.Event{Type: "notification.new", Data: *n})
	}
}

func (s *Service) List(ctx context.Context, userID, cursor string, limit int) (ListResponse, error) {
	var before *time.Time
	if cursor != "" {
		t, err := time.Parse(time.RFC3339Nano, cursor)
		if err != nil {
			return ListResponse{}, platformerrors.New(http.StatusBadRequest, "validation", "invalid cursor")
		}
		before = &t
	}
	items, err := s.repo.List(ctx, userID, before, limit+1)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{}
	if len(items) > limit {
		items = items[:limit]
		resp.NextCursor = items[len(items)-1].CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	resp.Notifications = items
	if resp.Unread, err = s.repo.UnreadCount(ctx, userID); err != nil {
		return ListResponse{}, err
	}
	return resp, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	err := s.repo.MarkRead(ctx, userID, id)
	if errors.Is(err, ErrNotFound) {
		return platformerrors.New(http.StatusNotFound, "not_found", "notification not found")
	}
	return err
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
