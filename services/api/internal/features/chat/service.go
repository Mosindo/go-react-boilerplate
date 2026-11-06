package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/platform/realtime"
)

var ErrInvalidMessage = errors.New("message must contain between 1 and 2000 characters")

type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) error
}

type Service struct {
	repo     Repository
	notifier Notifier
	pub      realtime.Publisher
}

func NewService(repo Repository, notifier Notifier, pub realtime.Publisher) *Service {
	if pub == nil {
		pub = realtime.NopPublisher{}
	}
	return &Service{repo: repo, notifier: notifier, pub: pub}
}

// NormalizeBody trims the message and strips control characters (keeping newlines).
func NormalizeBody(body string) (string, error) {
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, body)
	cleaned = strings.TrimSpace(cleaned)
	if n := utf8.RuneCountInString(cleaned); n < 1 || n > MaxBodyRunes {
		return "", ErrInvalidMessage
	}
	return cleaned, nil
}

func clampPage(n int) int {
	if n <= 0 {
		return DefaultPage
	}
	if n > MaxPage {
		return MaxPage
	}
	return n
}

func (s *Service) Conversations(ctx context.Context, userID string, before *time.Time, limit int) (ConversationsResponse, error) {
	limit = clampPage(limit)
	items, err := s.repo.ListConversations(ctx, userID, before, limit+1)
	if err != nil {
		return ConversationsResponse{}, err
	}
	resp := ConversationsResponse{}
	if len(items) > limit {
		items = items[:limit]
		next := items[len(items)-1].ActivityAt
		resp.NextBefore = &next
	}
	resp.Conversations = items
	if resp.UnreadTotal, err = s.repo.UnreadTotal(ctx, userID); err != nil {
		return ConversationsResponse{}, err
	}
	return resp, nil
}

func (s *Service) Messages(ctx context.Context, conversationID, userID string, before *time.Time, limit int) (MessagesResponse, error) {
	return s.repo.ListMessages(ctx, conversationID, userID, before, clampPage(limit))
}

func (s *Service) Send(ctx context.Context, conversationID, senderID, body string) (Message, error) {
	cleaned, err := NormalizeBody(body)
	if err != nil {
		return Message{}, err
	}
	res, err := s.repo.Send(ctx, conversationID, senderID, cleaned)
	if err != nil {
		return Message{}, err
	}
	s.pub.Publish(res.RecipientID, realtime.Event{Type: "message.new", Data: res.Message})
	// One notification per burst: only when the recipient had nothing unread from us.
	if res.RecipientUnread == 0 {
		_ = s.notifier.Notify(ctx, res.RecipientID, notifications.KindMessage, res.SenderName, preview(cleaned),
			map[string]string{"conversationId": conversationID, "userId": senderID})
	}
	return res.Message, nil
}

func (s *Service) MarkRead(ctx context.Context, conversationID, userID string) error {
	peer, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	s.pub.Publish(peer, realtime.Event{Type: "message.read", Data: map[string]any{
		"conversationId": conversationID, "readAt": time.Now().UTC()}})
	return nil
}

func (s *Service) Hide(ctx context.Context, conversationID, userID string) error {
	return s.repo.Hide(ctx, conversationID, userID)
}

func preview(body string) string {
	runes := []rune(strings.Join(strings.Fields(body), " "))
	if len(runes) > 80 {
		return string(runes[:79]) + "…"
	}
	return string(runes)
}
