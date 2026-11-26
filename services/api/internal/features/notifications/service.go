package notifications

import (
	"context"
	"log"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

var ErrNotFound = platformerrors.NotFound("notification_not_found", "Notification introuvable.")

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

// Notify stores a notification and pushes it in real time. Failures are logged rather than
// returned: a notification must never make the triggering action (match, message) fail.
func (s *Service) Notify(ctx context.Context, in Input) {
	n, err := s.repo.Upsert(ctx, in)
	if err != nil {
		log.Printf(`{"event":"notification_failed","type":%q,"error":%q}`, in.Type, err.Error())
		return
	}
	s.publisher.Publish(in.UserID, realtime.Event{Type: "notification.created", Data: n})
}

func (s *Service) List(ctx context.Context, userID string, cursor *httpx.Cursor, limit int) (Page, error) {
	items, err := s.repo.List(ctx, userID, cursor, limit+1)
	if err != nil {
		return Page{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return Page{}, err
	}
	page := Page{Notifications: items, UnreadCount: unread}
	if len(items) > limit {
		page.Notifications = items[:limit]
		last := page.Notifications[limit-1]
		next := httpx.EncodeCursor(last.CreatedAt, last.ID)
		page.NextCursor = &next
	}
	if page.Notifications == nil {
		page.Notifications = []Notification{}
	}
	return page, nil
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	ok, err := s.repo.MarkRead(ctx, userID, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) MarkGroupRead(ctx context.Context, userID, groupKey string) {
	if err := s.repo.MarkGroupRead(ctx, userID, groupKey); err != nil {
		log.Printf(`{"event":"notification_mark_group_failed","error":%q}`, err.Error())
	}
}

// Forget deletes notifications that point to content which no longer exists (unmatch, block).
func (s *Service) Forget(ctx context.Context, groupKeys ...string) {
	if len(groupKeys) == 0 {
		return
	}
	if err := s.repo.DeleteGroups(ctx, groupKeys); err != nil {
		log.Printf(`{"event":"notification_forget_failed","error":%q}`, err.Error())
	}
}
