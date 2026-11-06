package notifications

import (
	"context"
	"encoding/json"
	"time"

	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	if pub == nil {
		pub = realtime.NopPublisher{}
	}
	return &Service{repo: repo, pub: pub}
}

// Notify stores a notification and pushes it to the user's live connections.
// Notifications are produced by other features; there is no public create API.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	n, err := s.repo.Create(ctx, userID, kind, title, body, raw)
	if err != nil {
		return err
	}
	s.pub.Publish(userID, realtime.Event{Type: "notification.new", Data: n})
	return nil
}

func (s *Service) List(ctx context.Context, userID string, before *time.Time, limit int) (ListResponse, error) {
	items, err := s.repo.List(ctx, userID, before, limit+1)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{}
	if len(items) > limit {
		items = items[:limit]
		next := items[len(items)-1].CreatedAt
		resp.NextBefore = &next
	}
	resp.Notifications = items
	if resp.UnreadCount, err = s.repo.UnreadCount(ctx, userID); err != nil {
		return ListResponse{}, err
	}
	return resp, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
