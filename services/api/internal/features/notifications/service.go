package notifications

import (
	"context"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

const (
	defaultLimit  = 20
	maxLimit      = 50
	maxTitleRunes = 160
	maxBodyRunes  = 1000
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.NopPublisher{}
	}
	return &Service{repo: repo, publisher: publisher}
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}

// Notify stores an in-app notification and pushes it in real time. It is
// only callable by server-side features: clients cannot forge notifications.
// Failures are logged, never surfaced to the action that triggered them.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) {
	if data == nil {
		data = map[string]string{}
	}
	n, err := s.repo.Create(ctx, Notification{
		UserID: userID,
		Type:   kind,
		Title:  truncate(title, maxTitleRunes),
		Body:   truncate(body, maxBodyRunes),
		Data:   data,
	})
	if err != nil {
		log.Printf(`{"event":"notification_create_failed","type":%q,"error":%q}`, kind, err.Error())
		return
	}
	s.publisher.Publish(userID, "notification.new", ToResponse(n))
}

func (s *Service) List(ctx context.Context, userID string, beforeAt *time.Time, beforeID string, limit int) ([]Notification, error) {
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}
	return s.repo.List(ctx, userID, beforeAt, beforeID, limit)
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func ToResponse(n Notification) NotificationResponse {
	return NotificationResponse{
		ID:        n.ID,
		Type:      n.Type,
		Title:     n.Title,
		Body:      n.Body,
		Data:      n.Data,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt,
		ReadAt:    n.ReadAt,
	}
}
