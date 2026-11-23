package notifications

import (
	"context"
	"encoding/json"
	"log"

	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (NotificationsResponse, error) {
	items, err := s.repo.List(ctx, userID, limit+1, offset)
	if err != nil {
		return NotificationsResponse{}, err
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return NotificationsResponse{}, err
	}
	return NotificationsResponse{Notifications: items, UnreadCount: unread, HasMore: hasMore}, nil
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// MarkConversationRead clears message notifications once a chat was opened.
func (s *Service) MarkConversationRead(ctx context.Context, userID, conversationID string) {
	if err := s.repo.MarkConversationRead(ctx, userID, conversationID); err != nil {
		log.Printf(`{"event":"notification_mark_conversation_error","error":%q}`, err.Error())
	}
}

// Notify creates an in-app notification and pushes it in realtime. Failures
// are logged and never break the business action that triggered them.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) {
	collapseKey := data["collapseKey"]
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	n, created, err := s.repo.Create(context.WithoutCancel(ctx), userID, kind, title, body, payload, collapseKey)
	if err != nil {
		log.Printf(`{"event":"notification_create_error","type":%q,"error":%q}`, kind, err.Error())
		return
	}
	if created && s.publisher != nil {
		s.publisher.Publish([]string{userID}, realtime.Event{Type: "notification.created", Data: n})
	}
}
