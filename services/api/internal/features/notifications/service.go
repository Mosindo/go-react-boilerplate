package notifications

import (
	"context"
	"log"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo   Repository
	events realtime.Publisher
}

func NewService(repo Repository, events realtime.Publisher) *Service {
	return &Service{repo: repo, events: events}
}

// Notify stores a notification and pushes it to the user's live connections.
func (s *Service) Notify(ctx context.Context, userID, kind, actorID string, data map[string]string) error {
	n, err := s.repo.Upsert(ctx, userID, kind, actorID, data)
	if err != nil {
		return err
	}
	s.events.Publish(userID, realtime.Event{"type": "notification.new", "notification": n})
	return nil
}

// NotifyBestEffort is for side effects that must not fail the main action.
func (s *Service) NotifyBestEffort(ctx context.Context, userID, kind, actorID string, data map[string]string) {
	if err := s.Notify(ctx, userID, kind, actorID, data); err != nil {
		log.Printf("notify %s for %s: %v", kind, userID, err)
	}
}

func (s *Service) List(ctx context.Context, userID string, cursor *httpx.TimeCursor, limit int) (ListResult, error) {
	items, err := s.repo.List(ctx, userID, cursor, limit+1)
	if err != nil {
		return ListResult{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResult{}, err
	}
	res := ListResult{UnreadCount: unread}
	if len(items) > limit {
		items = items[:limit]
		last := items[len(items)-1]
		next := httpx.EncodeTimeCursor(last.CreatedAt, last.ID)
		res.NextCursor = &next
	}
	res.Notifications = items
	return res, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if !profiles.IsUUID(id) {
		return platformerrors.NotFound("notification not found")
	}
	ok, err := s.repo.MarkRead(ctx, userID, id)
	if err != nil {
		return err
	}
	if !ok {
		return platformerrors.NotFound("notification not found")
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// MarkConversationRead clears the unread message notification of a conversation.
func (s *Service) MarkConversationRead(ctx context.Context, userID, conversationID string) {
	if err := s.repo.MarkConversationRead(ctx, userID, conversationID); err != nil {
		log.Printf("mark conversation notifications read: %v", err)
	}
}
