package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

var ErrNotificationNotFound = apperr.NotFound("notification not found")

// Notifier is the dependency other features use to notify a user.
type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string)
	MarkConversationRead(ctx context.Context, userID, conversationID string)
}

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (ListResponse, error) {
	if offset < 0 {
		offset = 0
	}
	items, err := s.repo.List(ctx, userID, limit+1, offset)
	if err != nil {
		return ListResponse{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{Notifications: items, UnreadCount: unread}
	if len(items) > limit {
		resp.Notifications = items[:limit]
		next := offset + limit
		resp.NextOffset = &next
	}
	return resp, nil
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

// Notify stores a notification and pushes it in real time. Failures are
// logged, never propagated: a notification must not fail the main action.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) {
	if data == nil {
		data = map[string]string{}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	n, created, err := s.repo.Create(ctx, userID, kind, title, body, raw, data["dedupeKey"])
	if err != nil {
		log.Printf(`{"event":"notification_create_failed","type":%q,"error":%q}`, kind, err.Error())
		return
	}
	if created {
		s.publisher.Publish(ctx, []string{userID}, realtime.Event{Type: "notification.created", Data: n})
	}
}

func (s *Service) MarkConversationRead(ctx context.Context, userID, conversationID string) {
	if err := s.repo.MarkReadByConversation(ctx, userID, conversationID); err != nil {
		log.Printf(`{"event":"notification_mark_conversation_failed","error":%q}`, err.Error())
	}
}
