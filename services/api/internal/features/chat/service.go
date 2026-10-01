package chat

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

type ProfileReader interface {
	ListPublic(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
}

type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) error
}

type Publisher interface {
	Publish(userID string, event realtime.Event)
}

type Service struct {
	repo     Repository
	profiles ProfileReader
	notifier Notifier
	pub      Publisher
}

func NewService(repo Repository, profiles ProfileReader, notifier Notifier, pub Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, notifier: notifier, pub: pub}
}

func (s *Service) ListConversations(ctx context.Context, userID string, limit, offset int) ([]ConversationSummary, int, error) {
	rows, err := s.repo.ListConversations(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.TotalUnread(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.OtherUserID)
	}
	list, err := s.profiles.ListPublic(ctx, userID, ids)
	if err != nil {
		return nil, 0, err
	}
	byID := make(map[string]profiles.PublicProfile, len(list))
	for _, p := range list {
		byID[p.UserID] = p
	}
	out := make([]ConversationSummary, 0, len(rows))
	for _, r := range rows {
		p, ok := byID[r.OtherUserID]
		if !ok {
			continue
		}
		out = append(out, ConversationSummary{ID: r.ID, MatchID: r.MatchID, User: p, LastMessage: r.Last, UnreadCount: r.Unread})
	}
	return out, total, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID, beforeID string, limit int) ([]Message, bool, error) {
	if _, err := s.repo.Participant(ctx, conversationID, userID); err != nil {
		return nil, false, err
	}
	if limit <= 0 {
		limit = DefaultMessagesPage
	}
	if limit > MaxMessagesPage {
		limit = MaxMessagesPage
	}
	rows, err := s.repo.ListMessages(ctx, conversationID, userID, beforeID, limit)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}
	// rows are newest-first; clients render oldest-first.
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, hasMore, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, rawBody string) (Message, error) {
	body := cleanBody(rawBody)
	if body == "" {
		return Message{}, httpx.Invalid("message cannot be empty")
	}
	if utf8.RuneCountInString(body) > MaxMessageRunes {
		return Message{}, httpx.Invalid("message is too long (2000 characters max)")
	}
	other, err := s.repo.Participant(ctx, conversationID, userID)
	if err != nil {
		return Message{}, err
	}
	msg, err := s.repo.CreateMessage(ctx, conversationID, userID, other, body)
	if err != nil {
		return Message{}, err
	}

	event := realtime.Event{Type: "message", Data: msg}
	s.pub.Publish(other, event)
	s.pub.Publish(userID, event) // keeps the sender's other devices in sync

	name := "Un match"
	if list, err := s.profiles.ListPublic(ctx, other, []string{userID}); err == nil && len(list) > 0 {
		name = list[0].FirstName
	}
	// Notification text never contains the message itself (lock-screen privacy).
	_ = s.notifier.Notify(ctx, other, "message", "Nouveau message", name+" vous a écrit.",
		map[string]string{"conversationId": conversationID, "userId": userID})
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (int, error) {
	other, err := s.repo.Participant(ctx, conversationID, userID)
	if err != nil {
		return 0, err
	}
	n, at, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return 0, err
	}
	if n > 0 {
		s.pub.Publish(other, realtime.Event{Type: "read", Data: map[string]string{
			"conversationId": conversationID, "readerId": userID, "readAt": at.Format(time.RFC3339Nano),
		}})
	}
	return n, nil
}

func (s *Service) Clear(ctx context.Context, userID, conversationID string) error {
	if _, err := s.repo.Participant(ctx, conversationID, userID); err != nil {
		return err
	}
	return s.repo.Clear(ctx, conversationID, userID)
}

// cleanBody trims and removes control characters other than newline and tab.
func cleanBody(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r == '\r' {
			continue
		}
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}
