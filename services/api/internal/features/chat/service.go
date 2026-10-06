package chat

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/platform/realtime"
)

type Notifier interface {
	Notify(ctx context.Context, n notifications.New)
}

type Service struct {
	repo     Repository
	notifier Notifier
	hub      *realtime.Hub
}

func NewService(repo Repository, notifier Notifier, hub *realtime.Hub) *Service {
	return &Service{repo: repo, notifier: notifier, hub: hub}
}

func (s *Service) ListMatches(ctx context.Context, userID string, limit, offset int) ([]MatchSummary, int, error) {
	if limit <= 0 {
		limit = defaultMatchesLimit
	}
	if limit > maxMatchesLimit {
		limit = maxMatchesLimit
	}
	if offset < 0 {
		offset = 0
	}
	items, err := s.repo.ListMatches(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	unread, err := s.repo.UnreadTotal(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	return items, unread, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, matchID, beforeID string, limit int) ([]Message, bool, *time.Time, error) {
	m, err := s.repo.Membership(ctx, matchID, userID)
	if err != nil {
		return nil, false, nil, err
	}
	if limit <= 0 {
		limit = defaultMessagesLimit
	}
	if limit > maxMessagesLimit {
		limit = maxMessagesLimit
	}
	msgs, err := s.repo.ListMessages(ctx, m, beforeID, limit)
	if err != nil {
		return nil, false, nil, err
	}
	hasMore := len(msgs) > limit
	if hasMore {
		msgs = msgs[:limit]
	}
	// Return oldest first so clients append in reading order.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, hasMore, m.OtherLastRead, nil
}

func (s *Service) Send(ctx context.Context, userID, matchID, body string) (Message, error) {
	clean, err := normalizeBody(body)
	if err != nil {
		return Message{}, err
	}
	m, err := s.repo.Membership(ctx, matchID, userID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.CreateMessage(ctx, m, userID, clean)
	if err != nil {
		return Message{}, err
	}

	s.hub.Publish(m.OtherUserID, realtime.Event{Type: "message.new", Data: ToMessageResponse(msg)})
	s.notifier.Notify(ctx, notifications.New{
		UserID: m.OtherUserID,
		Type:   notifications.TypeNewMessage,
		Title:  "Nouveau message",
		Body:   m.SelfName + " vous a écrit.",
		Data:   map[string]string{"matchId": m.MatchID, "userId": userID},
		Dedupe: true,
	})
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, matchID string) error {
	m, err := s.repo.Membership(ctx, matchID, userID)
	if err != nil {
		return err
	}
	readAt, err := s.repo.MarkRead(ctx, m, userID)
	if err != nil {
		return err
	}
	s.hub.Publish(m.OtherUserID, realtime.Event{
		Type: "message.read",
		Data: map[string]any{"matchId": m.MatchID, "readAt": readAt},
	})
	return nil
}

// Hide removes the conversation from the caller's list only ("local delete").
func (s *Service) Hide(ctx context.Context, userID, matchID string) error {
	m, err := s.repo.Membership(ctx, matchID, userID)
	if err != nil {
		return err
	}
	return s.repo.Hide(ctx, m, userID)
}

// Unmatch ends the match for both people; the conversation becomes inaccessible.
func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	m, err := s.repo.Membership(ctx, matchID, userID)
	if err != nil {
		return err
	}
	if err := s.repo.Unmatch(ctx, m); err != nil {
		return err
	}
	s.hub.Publish(m.OtherUserID, realtime.Event{Type: "match.ended", Data: map[string]string{"matchId": m.MatchID}})
	return nil
}

func normalizeBody(raw string) (string, error) {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	body := strings.TrimSpace(b.String())
	if body == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(body) > maxMessageRunes {
		return "", ErrMessageTooBig
	}
	return body, nil
}
