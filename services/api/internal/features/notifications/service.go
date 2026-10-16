package notifications

import (
	"context"
	"strings"
	"unicode/utf8"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, n NewNotification) (Notification, error) {
	n.Title = truncate(n.Title, 120)
	n.Body = truncate(n.Body, 240)
	if n.Data == nil {
		n.Data = map[string]any{}
	}
	if n.Type == TypeMessage {
		if matchID, ok := n.Data["matchId"].(string); ok && matchID != "" {
			return s.repo.UpsertMessage(ctx, n, matchID)
		}
	}
	return s.repo.Create(ctx, n)
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) ([]Notification, error) {
	return s.repo.List(ctx, userID, limit, offset)
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	return s.repo.MarkRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// MarkConversationRead clears message notifications once the conversation was opened.
func (s *Service) MarkConversationRead(ctx context.Context, userID, matchID string) error {
	return s.repo.MarkConversationRead(ctx, userID, matchID)
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func truncate(v string, max int) string {
	v = strings.TrimSpace(v)
	if utf8.RuneCountInString(v) <= max {
		return v
	}
	r := []rune(v)
	return string(r[:max-1]) + "…"
}
