package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

var ErrNotificationNotFound = errors.New("notification not found")

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
	if err != nil {
		return nil, 0, err
	}
	return list, unread, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) error {
	if err := s.repo.MarkRead(ctx, userID, notificationID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrNotificationNotFound
		}
		return err
	}
	return nil
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) create(ctx context.Context, userID, kind, title, body string, data map[string]string, dedupeKey *string) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	n, err := s.repo.Upsert(ctx, userID, kind, title, body, raw, dedupeKey)
	if err != nil {
		// Notifications are best-effort: never fail the business action.
		log.Printf(`{"event":"notification_create_failed","user_id":%q,"type":%q,"error":%q}`, userID, kind, err.Error())
		return
	}
	s.publisher.Publish(userID, realtime.Event{Type: "notification.created", Data: toResponse(n)})
}

// NotifyMatch tells a user they have a new match.
func (s *Service) NotifyMatch(ctx context.Context, userID, otherFirstName, matchID, conversationID string) {
	s.create(ctx, userID, TypeMatch, "C'est un match !", "Toi et "+otherFirstName+" vous plaisez mutuellement.",
		map[string]string{"matchId": matchID, "conversationId": conversationID}, nil)
}

// NotifyMessage collapses unread message notifications per conversation.
func (s *Service) NotifyMessage(ctx context.Context, userID, senderFirstName, conversationID, preview string) {
	key := "message:" + conversationID
	s.create(ctx, userID, TypeMessage, "Nouveau message de "+senderFirstName, truncate(preview, 90),
		map[string]string{"conversationId": conversationID}, &key)
}

// MarkConversationRead clears message notifications once a conversation is read.
func (s *Service) MarkConversationRead(ctx context.Context, userID, conversationID string) {
	if err := s.repo.MarkReadByDedupeKey(ctx, userID, "message:"+conversationID); err != nil {
		log.Printf(`{"event":"notification_mark_read_failed","user_id":%q,"error":%q}`, userID, err.Error())
	}
}

func truncate(value string, maxRunes int) string {
	if utf8.RuneCountInString(value) <= maxRunes {
		return value
	}
	runes := []rune(value)
	return string(runes[:maxRunes-1]) + "…"
}
