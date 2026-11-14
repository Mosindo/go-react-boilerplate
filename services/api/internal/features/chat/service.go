package chat

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var ErrInvalidMessage = errors.New("invalid message")

type CardProvider interface {
	CardsByID(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.PublicProfile, error)
}

type MessageNotifier interface {
	NotifyMessage(ctx context.Context, userID, matchID, senderFirstName, preview string)
	MarkConversationRead(ctx context.Context, userID, matchID string) error
}

type Service struct {
	repo     Repository
	cards    CardProvider
	notifier MessageNotifier
	pub      realtime.Publisher
}

func NewService(repo Repository, cards CardProvider, notifier MessageNotifier, pub realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, notifier: notifier, pub: pub}
}

// NormalizeMessage trims the body, strips control characters (except new
// lines and tabs) and enforces 1..2000 characters.
func NormalizeMessage(body string) (string, error) {
	var b strings.Builder
	for _, r := range strings.TrimSpace(body) {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			continue
		}
		b.WriteRune(r)
	}
	normalized := strings.TrimSpace(b.String())
	if normalized == "" || utf8.RuneCountInString(normalized) > maxMessageRunes {
		return "", ErrInvalidMessage
	}
	return normalized, nil
}

func (s *Service) Conversations(ctx context.Context, userID string, limit, offset int) ([]Conversation, error) {
	rows, err := s.repo.ListConversations(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.OtherUserID)
	}
	cards, err := s.cards.CardsByID(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Conversation, 0, len(rows))
	for _, r := range rows {
		card, ok := cards[r.OtherUserID]
		if !ok {
			continue
		}
		out = append(out, Conversation{
			MatchID:         r.MatchID,
			User:            card,
			CreatedAt:       r.CreatedAt,
			LastMessage:     r.LastMessage,
			UnreadCount:     r.UnreadCount,
			OtherLastReadAt: r.OtherLastReadAt,
		})
	}
	return out, nil
}

func (s *Service) Messages(ctx context.Context, userID, matchID, beforeID string, limit int) (MessagesResponse, error) {
	m, err := s.repo.Membership(ctx, matchID, userID)
	if err != nil {
		return MessagesResponse{}, err
	}
	msgs, err := s.repo.ListMessages(ctx, matchID, m.ClearedAt, beforeID, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	hasMore := len(msgs) > limit
	if hasMore {
		msgs = msgs[:limit]
	}
	return MessagesResponse{Messages: msgs, HasMore: hasMore, OtherLastReadAt: m.OtherLastReadAt}, nil
}

func (s *Service) Send(ctx context.Context, userID, matchID, body string) (Message, error) {
	normalized, err := NormalizeMessage(body)
	if err != nil {
		return Message{}, err
	}
	m, err := s.repo.Membership(ctx, matchID, userID)
	if err != nil {
		return Message{}, err
	}
	msg, senderName, err := s.repo.CreateMessage(ctx, matchID, userID, normalized)
	if err != nil {
		return Message{}, err
	}
	if s.pub != nil {
		event := realtime.Event{Type: "message.created", Data: msg}
		s.pub.Publish(m.OtherUserID, event)
		s.pub.Publish(userID, event)
	}
	s.notifier.NotifyMessage(ctx, m.OtherUserID, matchID, senderName, normalized)
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, matchID string) (time.Time, error) {
	m, err := s.repo.Membership(ctx, matchID, userID)
	if err != nil {
		return time.Time{}, err
	}
	readAt, err := s.repo.MarkRead(ctx, matchID, userID)
	if err != nil {
		return time.Time{}, err
	}
	if err := s.notifier.MarkConversationRead(ctx, userID, matchID); err != nil {
		return time.Time{}, err
	}
	if s.pub != nil {
		s.pub.Publish(m.OtherUserID, realtime.Event{Type: "conversation.read", Data: map[string]any{"matchId": matchID, "readAt": readAt}})
	}
	return readAt, nil
}

// Clear hides the current history for the caller only ("delete for me").
func (s *Service) Clear(ctx context.Context, userID, matchID string) error {
	return s.repo.Clear(ctx, matchID, userID)
}
