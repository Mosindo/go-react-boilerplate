package notifications

import (
	"context"
	"log"

	"example.com/api/internal/platform/realtime"
)

const (
	defaultLimit = 30
	maxLimit     = 100
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (ListResponse, error) {
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}
	items, err := s.repo.List(ctx, userID, limit+1, offset)
	if err != nil {
		return ListResponse{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResponse{}, err
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	if items == nil {
		items = []Notification{}
	}
	return ListResponse{Notifications: items, UnreadCount: unread, HasMore: hasMore}, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) (Notification, error) {
	return s.repo.MarkRead(ctx, userID, id)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) publish(userID string, n Notification) {
	if s.publisher != nil {
		s.publisher.Publish(userID, realtime.Event{Type: "notification.created", Data: n})
	}
}

func displayName(name string) string {
	if name == "" {
		return "Quelqu'un"
	}
	return name
}

// NotifyMatch records a match notification for userID. Failures are logged,
// never propagated: a notification must not undo a match.
func (s *Service) NotifyMatch(ctx context.Context, userID, actorID, matchID string) {
	name, err := s.repo.FirstName(ctx, actorID)
	if err != nil {
		log.Printf(`{"event":"notify_match_failed","error":%q}`, err.Error())
		return
	}
	n, err := s.repo.CreateMatch(ctx, userID, actorID, matchID, "Nouveau match", "Toi et "+displayName(name)+" vous plaisez mutuellement.")
	if err != nil {
		log.Printf(`{"event":"notify_match_failed","error":%q}`, err.Error())
		return
	}
	s.publish(userID, n)
}

// NotifyMessage never includes the message content, so lock-screen or shared
// device previews do not leak conversations.
func (s *Service) NotifyMessage(ctx context.Context, recipientID, senderID, matchID string) {
	name, err := s.repo.FirstName(ctx, senderID)
	if err != nil {
		log.Printf(`{"event":"notify_message_failed","error":%q}`, err.Error())
		return
	}
	n, err := s.repo.UpsertMessage(ctx, recipientID, senderID, matchID, "Nouveau message", displayName(name)+" t'a écrit.")
	if err != nil {
		log.Printf(`{"event":"notify_message_failed","error":%q}`, err.Error())
		return
	}
	s.publish(recipientID, n)
}

func (s *Service) ConversationRead(ctx context.Context, userID, matchID string) {
	if err := s.repo.MarkMatchMessagesRead(ctx, userID, matchID); err != nil {
		log.Printf(`{"event":"notify_read_failed","error":%q}`, err.Error())
	}
}
