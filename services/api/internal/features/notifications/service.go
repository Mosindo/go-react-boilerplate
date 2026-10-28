package notifications

import (
	"context"
	"errors"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

var errNotFound = platformerrors.NotFound("not_found", "notification not found")

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

// Notify stores a notification and pushes it live. A nil result means it was deduplicated.
func (s *Service) Notify(ctx context.Context, userID, actorID, matchID, kind string) (*Notification, error) {
	n, err := s.repo.Create(ctx, userID, actorID, matchID, kind)
	if err != nil || n == nil {
		return n, err
	}
	s.pub.Publish(userID, realtime.Event{Type: "notification", Data: n})
	return n, nil
}

func (s *Service) MarkConversationRead(ctx context.Context, userID, matchID string) error {
	return s.repo.MarkConversationRead(ctx, userID, matchID)
}

func (s *Service) List(ctx context.Context, userID string, limit int, rawCursor string) (Page, error) {
	cursor, err := httpx.DecodeCursor(rawCursor)
	if err != nil {
		return Page{}, err
	}
	items, err := s.repo.List(ctx, userID, limit+1, cursor)
	if err != nil {
		return Page{}, err
	}
	page := Page{Notifications: items}
	if len(items) > limit {
		page.Notifications = items[:limit]
		last := page.Notifications[limit-1]
		next := httpx.EncodeCursor(last.CreatedAt, last.ID)
		page.NextCursor = &next
	}
	if page.Notifications == nil {
		page.Notifications = []Notification{}
	}
	page.UnreadCount, err = s.repo.UnreadCount(ctx, userID)
	return page, err
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if err := s.repo.MarkRead(ctx, userID, id); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return errNotFound
		}
		return err
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
