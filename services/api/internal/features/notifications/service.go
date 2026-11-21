package notifications

import (
	"context"
	"errors"
	"log"
	"strings"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

var (
	ErrTypeRequired  = errors.New("notification type required")
	ErrTitleRequired = errors.New("notification title required")
	ErrBodyRequired  = errors.New("notification body required")
	ErrTypeTooLong   = errors.New("notification type too long")
	ErrTitleTooLong  = errors.New("notification title too long")
	ErrBodyTooLong   = errors.New("notification body too long")
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

const (
	defaultNotificationsLimit = 20
	maxNotificationsLimit     = 100
	maxNotificationTypeRunes  = 64
	maxNotificationTitleRunes = 160
	maxNotificationBodyRunes  = 1000
)

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	if publisher == nil {
		publisher = realtime.Nop{}
	}
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) ListByUser(ctx context.Context, userID string, limit, offset int) ([]Notification, int, error) {
	normalizedLimit := normalizeNotificationsLimit(limit)
	normalizedOffset := normalizeNotificationsOffset(offset)
	stored, err := s.repo.ListNotifications(ctx, userID, normalizedLimit, normalizedOffset)
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.repo.CountUnread(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	payload := make([]Notification, 0, len(stored))
	for _, notification := range stored {
		payload = append(payload, mapStoredNotification(notification))
	}
	return payload, unread, nil
}

// Create stores a user-authored notification for the caller (generic module API).
func (s *Service) Create(ctx context.Context, userID, kind, title, body string) (Notification, error) {
	normalizedType, normalizedTitle, normalizedBody, err := validate(kind, title, body)
	if err != nil {
		return Notification{}, err
	}
	stored, err := s.repo.CreateNotification(ctx, userID, normalizedType, normalizedTitle, normalizedBody, nil)
	if err != nil {
		return Notification{}, err
	}
	n := mapStoredNotification(stored)
	s.push(n)
	return n, nil
}

// Notify creates a server-generated notification and pushes it in realtime.
// Failures are logged but never break the business action that triggered it.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) {
	stored, err := s.repo.CreateNotification(ctx, userID, kind, title, body, data)
	if err != nil {
		log.Printf(`{"event":"notification_create_failed","type":%q,"error":%q}`, kind, err.Error())
		return
	}
	s.push(mapStoredNotification(stored))
}

// NotifyCoalesced refreshes an existing unread notification with the same
// reference instead of creating a new one (used for chat messages).
func (s *Service) NotifyCoalesced(ctx context.Context, userID, kind, refKey, title, body string, data map[string]string) {
	stored, err := s.repo.UpsertUnreadByRef(ctx, userID, kind, refKey, data[refKey], title, body, data)
	if err != nil {
		log.Printf(`{"event":"notification_upsert_failed","type":%q,"error":%q}`, kind, err.Error())
		return
	}
	s.push(mapStoredNotification(stored))
}

// MarkReadByRef marks notifications tied to a reference as read (e.g. when a
// conversation is opened).
func (s *Service) MarkReadByRef(ctx context.Context, userID, kind, refKey, refValue string) {
	if err := s.repo.MarkReadByRef(ctx, userID, kind, refKey, refValue); err != nil {
		log.Printf(`{"event":"notification_mark_ref_failed","type":%q,"error":%q}`, kind, err.Error())
	}
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	notification, err := s.repo.MarkRead(ctx, userID, notificationID)
	if err != nil {
		if errors.Is(err, ErrNotificationNotFound) {
			return Notification{}, ErrNotificationNotFound
		}
		return Notification{}, err
	}
	return mapStoredNotification(notification), nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) push(n Notification) {
	s.publisher.Publish(n.UserID, realtime.Event{Type: "notification.new", Data: mapNotificationResponse(n)})
}

func validate(kind, title, body string) (string, string, string, error) {
	normalizedType := strings.TrimSpace(kind)
	if normalizedType == "" {
		return "", "", "", ErrTypeRequired
	}
	if utf8.RuneCountInString(normalizedType) > maxNotificationTypeRunes {
		return "", "", "", ErrTypeTooLong
	}
	normalizedTitle := strings.TrimSpace(title)
	if normalizedTitle == "" {
		return "", "", "", ErrTitleRequired
	}
	if utf8.RuneCountInString(normalizedTitle) > maxNotificationTitleRunes {
		return "", "", "", ErrTitleTooLong
	}
	normalizedBody := strings.TrimSpace(body)
	if normalizedBody == "" {
		return "", "", "", ErrBodyRequired
	}
	if utf8.RuneCountInString(normalizedBody) > maxNotificationBodyRunes {
		return "", "", "", ErrBodyTooLong
	}
	return normalizedType, normalizedTitle, normalizedBody, nil
}

func mapStoredNotification(notification StoredNotification) Notification {
	return Notification{
		ID:        notification.ID,
		UserID:    notification.UserID,
		Type:      notification.Type,
		Title:     notification.Title,
		Body:      notification.Body,
		Data:      notification.Data,
		IsRead:    notification.IsRead,
		CreatedAt: notification.CreatedAt,
		ReadAt:    notification.ReadAt,
	}
}

func normalizeNotificationsLimit(limit int) int {
	if limit <= 0 {
		return defaultNotificationsLimit
	}
	if limit > maxNotificationsLimit {
		return maxNotificationsLimit
	}
	return limit
}

func normalizeNotificationsOffset(offset int) int {
	if offset < 0 {
		return 0
	}
	return offset
}
