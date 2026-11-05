package notifications

import (
	"context"
	"encoding/base64"
	"strings"
	"time"
	"unicode/utf8"
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) NotifyMatch(ctx context.Context, userID, otherName, conversationID string) (Notification, error) {
	return s.repo.Create(ctx, userID, "match", "Nouveau match", "Vous avez un match avec "+otherName+" !", &conversationID)
}

func (s *Service) NotifyMessage(ctx context.Context, userID, senderName, conversationID, body string) (Notification, error) {
	return s.repo.UpsertMessage(ctx, userID, conversationID, senderName, Preview(body))
}

func (s *Service) MarkConversationRead(ctx context.Context, userID, conversationID string) error {
	return s.repo.MarkConversationRead(ctx, userID, conversationID)
}

func (s *Service) List(ctx context.Context, userID string, limit int, cursor string) (Page, error) {
	var before *time.Time
	var beforeID string
	if cursor != "" {
		t, id, ok := decodeCursor(cursor)
		if !ok {
			return Page{}, ErrBadCursor
		}
		before, beforeID = &t, id
	}
	items, err := s.repo.List(ctx, userID, limit+1, before, beforeID)
	if err != nil {
		return Page{}, err
	}
	page := Page{Notifications: items}
	if len(items) > limit {
		page.Notifications = items[:limit]
		last := page.Notifications[limit-1]
		page.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	return page, nil
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}
func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	return s.repo.MarkRead(ctx, userID, id)
}
func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// Preview shortens a message body for display in a notification.
func Preview(body string) string {
	body = strings.Join(strings.Fields(body), " ")
	if utf8.RuneCountInString(body) <= 80 {
		return body
	}
	return string([]rune(body)[:79]) + "…"
}

func encodeCursor(t time.Time, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(t.UTC().Format(time.RFC3339Nano) + "|" + id))
}

func decodeCursor(c string) (time.Time, string, bool) {
	raw, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, "", false
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", false
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", false
	}
	return t, parts[1], true
}
