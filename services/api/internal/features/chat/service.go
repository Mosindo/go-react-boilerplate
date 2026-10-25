package chat

import (
	"context"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

type SummaryReader interface {
	Summaries(ctx context.Context, userIDs []string) (map[string]profiles.ProfileSummary, error)
}

type NotificationPusher interface {
	Push(ctx context.Context, userID, notificationID string)
}

type Service struct {
	repo     *Repository
	profiles SummaryReader
	events   realtime.Publisher
	notifier NotificationPusher
}

func NewService(repo *Repository, profiles SummaryReader, events realtime.Publisher, notifier NotificationPusher) *Service {
	return &Service{repo: repo, profiles: profiles, events: events, notifier: notifier}
}

func (s *Service) ListConversations(ctx context.Context, userID string, limit int, cur *httpx.Cursor) (httpx.Page[ConversationItem], error) {
	rows, err := s.repo.listConversations(ctx, userID, limit, cur)
	if err != nil {
		return httpx.Page[ConversationItem]{}, err
	}
	page := httpx.Paginate(rows, limit, func(r conversationRow) httpx.Cursor { return httpx.Cursor{At: r.UpdatedAt, ID: r.ID} })

	ids := make([]string, len(page.Items))
	for i, r := range page.Items {
		ids[i] = r.OtherUserID
	}
	summaries, err := s.profiles.Summaries(ctx, ids)
	if err != nil {
		return httpx.Page[ConversationItem]{}, err
	}
	items := make([]ConversationItem, 0, len(page.Items))
	for _, r := range page.Items {
		last := r.Last
		items = append(items, ConversationItem{
			ID: r.ID, MatchID: r.MatchID, OtherUser: summaries[r.OtherUserID], LastMessage: &last,
			UnreadCount: r.UnreadCount, UpdatedAt: r.UpdatedAt,
		})
	}
	return httpx.Page[ConversationItem]{Items: items, NextCursor: page.NextCursor}, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID string, limit int, cur *httpx.Cursor) (httpx.Page[Message], error) {
	rows, err := s.repo.listMessages(ctx, userID, conversationID, limit, cur)
	if err != nil {
		return httpx.Page[Message]{}, err
	}
	return httpx.Paginate(rows, limit, func(m Message) httpx.Cursor { return httpx.Cursor{At: m.CreatedAt, ID: m.ID} }), nil
}

func normalizeBody(raw string) (string, error) {
	body := strings.TrimSpace(raw)
	if body == "" || !utf8.ValidString(body) || utf8.RuneCountInString(body) > MaxMessageRunes {
		return "", ErrInvalidMessage
	}
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return "", ErrInvalidMessage
		}
	}
	return body, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, rawBody string) (Message, error) {
	body, err := normalizeBody(rawBody)
	if err != nil {
		return Message{}, err
	}
	res, err := s.repo.send(ctx, userID, conversationID, body)
	if err != nil {
		return Message{}, err
	}
	evt := realtime.Event{Type: "message", Data: res.Message}
	s.events.Publish(userID, evt)
	s.events.Publish(res.OtherUserID, evt)
	if res.NotificationID != "" {
		s.notifier.Push(ctx, res.OtherUserID, res.NotificationID)
	}
	return res.Message, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	res, err := s.repo.markRead(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	if res.Count > 0 {
		s.events.Publish(res.OtherUserID, realtime.Event{Type: "read", Data: map[string]any{
			"conversationId": conversationID, "readerId": userID, "readAt": res.ReadAt,
		}})
	}
	return nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	return s.repo.hide(ctx, userID, conversationID)
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.unreadCount(ctx, userID)
}

// PrepareAccountRemoval is an auth.AccountHook: once the account is gone, partners are told the
// conversations disappeared and the user's sockets are closed.
func (s *Service) PrepareAccountRemoval(ctx context.Context, userID string) (func(), error) {
	partners, err := s.repo.partnersOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	return func() {
		for conversationID, other := range partners {
			s.events.Publish(other, realtime.Event{Type: "conversation_removed", Data: map[string]string{"conversationId": conversationID}})
		}
		s.events.Disconnect(userID)
	}, nil
}
