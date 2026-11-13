package notifications

import (
	"context"
	"log"

	"example.com/api/internal/platform/realtime"
)

const (
	defaultLimit = 20
	maxLimit     = 50
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.Nop{}
	}
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (NotificationsResponse, error) {
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}
	if offset < 0 {
		offset = 0
	}
	items, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return NotificationsResponse{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return NotificationsResponse{}, err
	}
	return NotificationsResponse{Notifications: items, UnreadCount: unread}, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// MarkConversationRead implements chat.ConversationReadHook.
func (s *Service) MarkConversationRead(ctx context.Context, userID, peerUserID string) {
	if err := s.repo.MarkPeerMessagesRead(ctx, userID, peerUserID); err != nil {
		log.Printf(`{"event":"notification_mark_read_failed","error":%q}`, err.Error())
	}
}

// Notify stores an in-app notification and pushes it over the realtime
// channel. Failures are logged, never propagated: a notification must not
// fail the action (match, message) that triggered it.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string, dedupeKey string) {
	created, inserted, err := s.repo.Create(ctx, Notification{
		UserID: userID,
		Type:   kind,
		Title:  title,
		Body:   body,
		Data:   data,
	}, dedupeKey)
	if err != nil {
		log.Printf(`{"event":"notification_create_failed","type":%q,"error":%q}`, kind, err.Error())
		return
	}
	if inserted {
		s.publisher.Publish(userID, "notification.created", created)
	}
}
