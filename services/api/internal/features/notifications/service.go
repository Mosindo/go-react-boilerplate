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

const (
	maxTypeRunes  = 64
	maxTitleRunes = 160
	maxBodyRunes  = 1000
)

type Service struct {
	repo   Repository
	events realtime.Publisher
}

func NewService(repo Repository, events realtime.Publisher) *Service {
	return &Service{repo: repo, events: events}
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (NotificationsResponse, error) {
	list, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return NotificationsResponse{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return NotificationsResponse{}, err
	}
	return NotificationsResponse{Notifications: list, UnreadCount: unread}, nil
}

func normalize(value string, max int, required, tooLong error) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", required
	}
	if utf8.RuneCountInString(v) > max {
		return "", tooLong
	}
	return v, nil
}

// Create validates and stores a notification addressed to userID.
func (s *Service) Create(ctx context.Context, userID, kind, title, body string, data map[string]string) (Notification, error) {
	kind, err := normalize(kind, maxTypeRunes, ErrTypeRequired, ErrTypeTooLong)
	if err != nil {
		return Notification{}, err
	}
	title, err = normalize(title, maxTitleRunes, ErrTitleRequired, ErrTitleTooLong)
	if err != nil {
		return Notification{}, err
	}
	body, err = normalize(body, maxBodyRunes, ErrBodyRequired, ErrBodyTooLong)
	if err != nil {
		return Notification{}, err
	}
	n, err := s.repo.Create(ctx, userID, kind, title, body, data)
	if err != nil {
		return Notification{}, err
	}
	s.events.Publish(userID, "notification.created", n)
	return n, nil
}

// Notify is the fire-and-forget variant used by other features: failures are
// logged and never break the business operation that triggered them.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) {
	if _, err := s.Create(ctx, userID, kind, title, body, data); err != nil {
		log.Printf(`{"event":"notification_failed","type":%q,"error":%q}`, kind, err.Error())
	}
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) (int, error) {
	return s.repo.MarkAllRead(ctx, userID)
}
