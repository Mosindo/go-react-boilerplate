package notifications

import (
	"context"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

const previewRunes = 80

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) ([]Notification, int, error) {
	list, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	return list, unread, err
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) publish(userID string, n Notification) {
	s.publisher.Publish(userID, realtime.Event{Type: "notification.new", Data: n})
}

// Notify creates an in-app notification and pushes it in real time.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) error {
	n, err := s.repo.Create(ctx, userID, kind, title, body, data)
	if err != nil {
		return err
	}
	s.publish(userID, n)
	return nil
}

// NotifyMessage keeps a single unread "new message" notification per conversation.
func (s *Service) NotifyMessage(ctx context.Context, recipientID, conversationID, senderID, senderName, preview string) error {
	data := map[string]string{"conversationId": conversationID, "userId": senderID}
	n, _, err := s.repo.RefreshUnread(ctx, recipientID, "message", "conversationId", conversationID,
		"Nouveau message de "+senderName, Truncate(preview, previewRunes), data)
	if err != nil {
		return err
	}
	s.publish(recipientID, n)
	return nil
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// Truncate shortens text to n runes, adding an ellipsis when cut.
func Truncate(text string, n int) string {
	if utf8.RuneCountInString(text) <= n {
		return text
	}
	runes := []rune(text)
	return string(runes[:n-1]) + "…"
}
