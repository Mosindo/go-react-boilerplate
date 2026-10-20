package notifications

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
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
	repo Repository
}

const (
	defaultNotificationsLimit = 20
	maxNotificationsLimit     = 100
	maxNotificationTypeRunes  = 64
	maxNotificationTitleRunes = 160
	maxNotificationBodyRunes  = 1000
)

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListByUser(ctx context.Context, userID string, limit, offset int) ([]Notification, error) {
	normalizedLimit := normalizeNotificationsLimit(limit)
	normalizedOffset := normalizeNotificationsOffset(offset)
	notifications, err := s.repo.ListNotifications(ctx, userID, normalizedLimit, normalizedOffset)
	if err != nil {
		return nil, err
	}

	payload := make([]Notification, 0, len(notifications))
	for _, notification := range notifications {
		payload = append(payload, mapStoredNotification(notification))
	}
	return payload, nil
}

func (s *Service) Create(ctx context.Context, userID, kind, title, body string) (Notification, error) {
	normalizedType := strings.TrimSpace(kind)
	if normalizedType == "" {
		return Notification{}, ErrTypeRequired
	}
	if utf8.RuneCountInString(normalizedType) > maxNotificationTypeRunes {
		return Notification{}, ErrTypeTooLong
	}
	normalizedTitle := strings.TrimSpace(title)
	if normalizedTitle == "" {
		return Notification{}, ErrTitleRequired
	}
	if utf8.RuneCountInString(normalizedTitle) > maxNotificationTitleRunes {
		return Notification{}, ErrTitleTooLong
	}
	normalizedBody := strings.TrimSpace(body)
	if normalizedBody == "" {
		return Notification{}, ErrBodyRequired
	}
	if utf8.RuneCountInString(normalizedBody) > maxNotificationBodyRunes {
		return Notification{}, ErrBodyTooLong
	}

	notification, err := s.repo.CreateNotification(ctx, userID, normalizedType, normalizedTitle, normalizedBody)
	if err != nil {
		return Notification{}, err
	}
	return mapStoredNotification(notification), nil
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

func mapStoredNotification(notification StoredNotification) Notification {
	return Notification{
		ID:        notification.ID,
		UserID:    notification.UserID,
		Type:      notification.Type,
		Title:     notification.Title,
		Body:      notification.Body,
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
