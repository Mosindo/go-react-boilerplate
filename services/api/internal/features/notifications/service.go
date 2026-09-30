package notifications

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"regexp"
	"sync"
	"time"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/push"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrNotificationNotFound = apperr.NotFound("notification not found")
	ErrInvalidPushToken     = apperr.Validation("invalid push token")
	pushTokenPattern        = regexp.MustCompile(`^Expo(nent)?PushToken\[[A-Za-z0-9_-]{10,100}\]$`)
)

const pushTimeout = 15 * time.Second

// Notifier is the dependency other features use to notify a user.
type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string)
	MarkConversationRead(ctx context.Context, userID, conversationID string)
}

type Service struct {
	repo      Repository
	publisher realtime.Publisher
	pusher    push.Sender
	// pushWG lets tests wait for asynchronous push deliveries.
	pushWG sync.WaitGroup
}

func NewService(repo Repository, publisher realtime.Publisher, pusher push.Sender) *Service {
	if pusher == nil {
		pusher = push.NopSender{}
	}
	return &Service{repo: repo, publisher: publisher, pusher: pusher}
}

func (s *Service) RegisterPushToken(ctx context.Context, userID, token, platform string) error {
	if !pushTokenPattern.MatchString(token) || (platform != "ios" && platform != "android") {
		return ErrInvalidPushToken
	}
	return s.repo.SavePushToken(ctx, userID, token, platform)
}

func (s *Service) UnregisterPushToken(ctx context.Context, userID, token string) error {
	return s.repo.DeletePushToken(ctx, userID, token)
}

// WaitForPushes blocks until in-flight push deliveries are done (tests).
func (s *Service) WaitForPushes() {
	s.pushWG.Wait()
}

// deliverPush sends a notification to the member's devices without blocking
// the request that triggered it. Dead tokens are removed.
func (s *Service) deliverPush(userID string, n Notification, data map[string]string) {
	s.pushWG.Add(1)
	go func() {
		defer s.pushWG.Done()
		ctx, cancel := context.WithTimeout(context.Background(), pushTimeout)
		defer cancel()
		tokens, err := s.repo.PushTokens(ctx, userID)
		if err != nil || len(tokens) == 0 {
			return
		}
		payload := map[string]string{"notificationId": n.ID, "type": n.Type}
		for _, key := range []string{"conversationId", "matchId"} {
			if v := data[key]; v != "" {
				payload[key] = v
			}
		}
		messages := make([]push.Message, 0, len(tokens))
		for _, token := range tokens {
			messages = append(messages, push.Message{To: token, Title: n.Title, Body: n.Body, Data: payload, Sound: "default", ChannelID: "default"})
		}
		results, err := s.pusher.Send(ctx, messages)
		if err != nil {
			log.Printf(`{"event":"push_send_failed","error":%q}`, err.Error())
			return
		}
		dead := []string{}
		for _, r := range results {
			if r.Unregistered {
				dead = append(dead, r.Token)
			}
		}
		if len(dead) > 0 {
			if err := s.repo.ForgetPushTokens(ctx, dead); err != nil {
				log.Printf(`{"event":"push_token_cleanup_failed","error":%q}`, err.Error())
			}
		}
	}()
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) (ListResponse, error) {
	if offset < 0 {
		offset = 0
	}
	items, err := s.repo.List(ctx, userID, limit+1, offset)
	if err != nil {
		return ListResponse{}, err
	}
	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResponse{}, err
	}
	resp := ListResponse{Notifications: items, UnreadCount: unread}
	if len(items) > limit {
		resp.Notifications = items[:limit]
		next := offset + limit
		resp.NextOffset = &next
	}
	return resp, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	n, err := s.repo.MarkRead(ctx, userID, notificationID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return Notification{}, ErrNotificationNotFound
	}
	return n, err
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllRead(ctx, userID)
}

// Notify stores a notification and pushes it in real time. Failures are
// logged, never propagated: a notification must not fail the main action.
func (s *Service) Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) {
	if data == nil {
		data = map[string]string{}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	n, created, err := s.repo.Create(ctx, userID, kind, title, body, raw, data["dedupeKey"])
	if err != nil {
		log.Printf(`{"event":"notification_create_failed","type":%q,"error":%q}`, kind, err.Error())
		return
	}
	if created {
		s.publisher.Publish(ctx, []string{userID}, realtime.Event{Type: "notification.created", Data: n})
		s.deliverPush(userID, n, data)
	}
}

func (s *Service) MarkConversationRead(ctx context.Context, userID, conversationID string) {
	if err := s.repo.MarkReadByConversation(ctx, userID, conversationID); err != nil {
		log.Printf(`{"event":"notification_mark_conversation_failed","error":%q}`, err.Error())
	}
}
