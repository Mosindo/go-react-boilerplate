package notifications

import (
	"context"
	"encoding/json"
	"log"

	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	pub  realtime.Publisher
}

func NewService(repo Repository, pub realtime.Publisher) *Service {
	return &Service{repo: repo, pub: pub}
}

// Notify stores an in-app notification and pushes it in real time. Failures
// are logged and never break the business action that triggered them.
func (s *Service) Notify(ctx context.Context, in Input) {
	data, err := json.Marshal(in.Data)
	if err != nil {
		data = []byte("{}")
	}
	n, err := s.repo.Upsert(ctx, in, data)
	if err != nil {
		log.Printf(`{"event":"notification_failed","user_id":%q,"type":%q,"error":%q}`, in.UserID, in.Type, err.Error())
		return
	}
	s.pub.Publish(in.UserID, realtime.Event{Type: realtime.EventNotification, Data: n})
}

// ClearConversation marks the "new messages" notification of a conversation as read.
func (s *Service) ClearConversation(ctx context.Context, userID, matchID string) {
	if err := s.repo.MarkReadByDedupe(ctx, userID, MessageDedupeKey(matchID)); err != nil {
		log.Printf(`{"event":"notification_clear_failed","user_id":%q,"error":%q}`, userID, err.Error())
	}
}

func MessageDedupeKey(matchID string) string { return "message:" + matchID }

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (ListResponse, error) {
	items, err := s.repo.List(ctx, userID, limit+1, offset)
	if err != nil {
		return ListResponse{}, err
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Notifications: items, UnreadCount: unread, HasMore: hasMore}, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}
