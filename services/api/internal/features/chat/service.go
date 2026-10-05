package chat

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrEmptyMessage   = errors.New("message cannot be empty")
	ErrMessageTooLong = errors.New("message is too long (2000 characters max)")
)

type ProfileReader interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Card, error)
}

type Service struct {
	repo     Repository
	profiles ProfileReader
	events   realtime.Publisher
}

func NewService(repo Repository, profiles ProfileReader, events realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, events: events}
}

func limitOr(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	return min(limit, max)
}

func (s *Service) previews(ctx context.Context, userID string, rows []convRow) ([]ConversationPreview, error) {
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.OtherUserID)
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]ConversationPreview, 0, len(rows))
	for _, r := range rows {
		card, ok := cards[r.OtherUserID]
		if !ok {
			continue
		}
		out = append(out, ConversationPreview{ID: r.ID, MatchID: r.MatchID, User: card, LastMessage: r.Last, UnreadCount: r.Unread})
	}
	return out, nil
}

func (s *Service) ListConversations(ctx context.Context, userID string, limit, offset int) ([]ConversationPreview, error) {
	rows, err := s.repo.ListConversations(ctx, userID, limitOr(limit, defaultConversationsLimit, maxConversationsLimit), max(offset, 0))
	if err != nil {
		return nil, err
	}
	return s.previews(ctx, userID, rows)
}

func (s *Service) Messages(ctx context.Context, userID, conversationID, before string, limit int) (MessagesResponse, error) {
	msgs, more, err := s.repo.ListMessages(ctx, userID, conversationID, before, limitOr(limit, defaultMessagesLimit, maxMessagesLimit))
	if err != nil {
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{Messages: make([]Message, 0, len(msgs))}
	if more && len(msgs) > 0 {
		resp.NextCursor = msgs[len(msgs)-1].ID
	}
	for i := len(msgs) - 1; i >= 0; i-- { // newest-first from SQL → chronological for display
		resp.Messages = append(resp.Messages, msgs[i])
	}
	return resp, nil
}

// normalizeContent trims, strips control characters (keeping newlines) and enforces the length limits.
func normalizeContent(raw string) (string, error) {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	content := strings.TrimSpace(b.String())
	if content == "" {
		return "", ErrEmptyMessage
	}
	if utf8.RuneCountInString(content) > MaxMessageRunes {
		return "", ErrMessageTooLong
	}
	return content, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, raw string) (Message, error) {
	content, err := normalizeContent(raw)
	if err != nil {
		return Message{}, err
	}
	res, err := s.repo.SendMessage(ctx, userID, conversationID, content)
	if err != nil {
		return Message{}, err
	}
	event := realtime.Event{Type: "message.new", Data: res.Message}
	s.events.Publish(res.RecipientID, event)
	s.events.Publish(userID, event) // keep the sender's other devices in sync
	if res.Notification != nil {
		s.events.Publish(res.RecipientID, realtime.Event{Type: "notification.new", Data: res.Notification})
	}
	return res.Message, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	readAt, changed, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	if changed > 0 {
		other, err := s.repo.Access(ctx, userID, conversationID)
		if err == nil {
			s.events.Publish(other, realtime.Event{Type: "messages.read", Data: map[string]any{
				"conversationId": conversationID, "readerUserId": userID, "readAt": readAt,
			}})
		}
	}
	return nil
}

func (s *Service) Clear(ctx context.Context, userID, conversationID string) error {
	return s.repo.Clear(ctx, userID, conversationID)
}
