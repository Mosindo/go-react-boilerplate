package notifications

import (
	"context"
	"log"

	"example.com/api/internal/platform/apperr"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

var ErrNotFound = apperr.NotFound("notification_not_found", "notification not found")

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

// Notify stores a notification and pushes it in realtime. Failures are logged
// and never break the business action that triggered them.
func (s *Service) Notify(ctx context.Context, in Input) {
	n, err := s.repo.Upsert(ctx, in)
	if err != nil {
		log.Printf(`{"event":"notification_error","type":%q,"error":%q}`, in.Type, err.Error())
		return
	}
	if s.publisher != nil {
		s.publisher.Publish(in.UserID, realtime.Event{Type: "notification.new", Data: n})
	}
}

func (s *Service) List(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) (ListResponse, error) {
	items, err := s.repo.List(ctx, userID, cursor, limit+1)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{Notifications: items}
	if len(items) > limit {
		resp.Notifications = items[:limit]
		last := resp.Notifications[limit-1]
		next := httpx.EncodeCursor(last.CreatedAt, last.ID)
		resp.NextCursor = &next
	}
	return resp, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// MarkGroupRead is used when the user opens the related screen (e.g. reads a
// conversation).
func (s *Service) MarkGroupRead(ctx context.Context, userID, groupKey string) {
	if err := s.repo.MarkGroupRead(ctx, userID, groupKey); err != nil {
		log.Printf(`{"event":"notification_group_read_error","error":%q}`, err.Error())
	}
}

func (s *Service) Badges(ctx context.Context, userID string) (Badges, error) {
	return s.repo.Badges(ctx, userID)
}
