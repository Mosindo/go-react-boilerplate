package notifications

import (
	"context"
	"encoding/json"
	"log"
	"unicode/utf8"

	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (ListResponse, error) {
	items, err := s.repo.List(ctx, userID, limit, offset)
	if err != nil {
		return ListResponse{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Notifications: items, UnreadCount: unread}, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) MarkConversationRead(ctx context.Context, userID, matchID string) error {
	return s.repo.MarkConversationRead(ctx, userID, matchID)
}

func (s *Service) publish(userID string, n Notification) {
	if s.pub != nil {
		s.pub.Publish(userID, realtime.Event{Type: "notification.created", Data: n})
	}
}

// NotifyMatch records a "new match" notification. Failures are logged, not
// returned: a notification must never roll back the match itself.
func (s *Service) NotifyMatch(ctx context.Context, userID, matchID, otherUserID, otherFirstName string) {
	data, _ := json.Marshal(map[string]string{"matchId": matchID, "userId": otherUserID})
	n, err := s.repo.Create(ctx, userID, TypeMatch, "Nouveau match", "Vous et "+otherFirstName+" vous plaisez mutuellement. Dites bonjour !", data)
	if err != nil {
		log.Printf(`{"event":"notification_failed","type":"match","error":%q}`, err.Error())
		return
	}
	s.publish(userID, n)
}

func (s *Service) NotifyMessage(ctx context.Context, userID, matchID, senderFirstName, preview string) {
	if utf8.RuneCountInString(preview) > 80 {
		preview = string([]rune(preview)[:80]) + "…"
	}
	data, _ := json.Marshal(map[string]string{"matchId": matchID})
	n, err := s.repo.UpsertMessage(ctx, userID, matchID, "Message de "+senderFirstName, preview, data)
	if err != nil {
		log.Printf(`{"event":"notification_failed","type":"message","error":%q}`, err.Error())
		return
	}
	s.publish(userID, n)
}
