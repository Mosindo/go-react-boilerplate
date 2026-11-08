package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/validate"
)

var (
	ErrConversationNotFound = platformerrors.ErrNotFound
	ErrBlocked              = platformerrors.New(403, "you can no longer exchange messages with this person")
)

type Service struct {
	repo     Repository
	cards    profiles.CardLoader
	notifier notifications.Notifier
	events   realtime.Publisher
}

func NewService(repo Repository, cards profiles.CardLoader, notifier notifications.Notifier, events realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, notifier: notifier, events: events}
}

// authorize returns the peer info or 404 — the same answer for "does not exist", "not yours" and
// "blocked", so conversation ids cannot be probed.
func (s *Service) authorize(ctx context.Context, conversationID, userID string) (participantInfo, error) {
	if !profiles.IsUUID(conversationID) {
		return participantInfo{}, ErrConversationNotFound
	}
	p, err := s.repo.Participant(ctx, conversationID, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return participantInfo{}, ErrConversationNotFound
		}
		return participantInfo{}, err
	}
	return p, nil
}

func (s *Service) ListConversations(ctx context.Context, userID string, before *time.Time, limit int) (ConversationsResponse, error) {
	rows, err := s.repo.ListConversations(ctx, userID, before, limit+1)
	if err != nil {
		return ConversationsResponse{}, err
	}
	var next *time.Time
	if len(rows) > limit {
		rows = rows[:limit]
		t := rows[len(rows)-1].SortAt
		next = &t
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.PeerID
	}
	cards, err := s.cards.LoadCards(ctx, userID, ids)
	if err != nil {
		return ConversationsResponse{}, err
	}
	byID := make(map[string]profiles.Card, len(cards))
	for _, c := range cards {
		byID[c.UserID] = c
	}
	out := make([]Conversation, 0, len(rows))
	for _, r := range rows {
		if c, ok := byID[r.PeerID]; ok {
			out = append(out, Conversation{ID: r.ID, Peer: c, LastMessage: r.LastMessage, UnreadCount: r.UnreadCount, PeerLastReadAt: r.PeerLastReadAt})
		}
	}
	return ConversationsResponse{Conversations: out, NextBefore: next}, nil
}

func (s *Service) ListMessages(ctx context.Context, userID, conversationID string, before *time.Time, limit int) (MessagesResponse, error) {
	p, err := s.authorize(ctx, conversationID, userID)
	if err != nil {
		return MessagesResponse{}, err
	}
	if p.Blocked {
		return MessagesResponse{}, ErrConversationNotFound
	}
	msgs, err := s.repo.ListMessages(ctx, conversationID, userID, before, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	var next *time.Time
	if len(msgs) > limit {
		msgs = msgs[:limit]
		t := msgs[len(msgs)-1].CreatedAt
		next = &t
	}
	return MessagesResponse{Messages: msgs, NextBefore: next, PeerLastReadAt: p.PeerLastReadAt}, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	body, ok := validate.CleanText(body, MaxMessageRunes, true)
	if !ok || body == "" {
		return Message{}, platformerrors.Invalid("message must contain 1 to 2000 characters")
	}
	p, err := s.authorize(ctx, conversationID, userID)
	if err != nil {
		return Message{}, err
	}
	if p.Blocked {
		return Message{}, ErrBlocked
	}
	msg, err := s.repo.InsertMessage(ctx, conversationID, userID, body)
	if err != nil {
		return Message{}, err
	}
	ev := realtime.Event{Type: "message.new", Data: msg}
	s.events.Publish(p.PeerID, ev)
	s.events.Publish(userID, ev) // the sender's other devices
	convID := conversationID
	s.notifier.Notify(ctx, p.PeerID, notifications.TypeMessage, &userID, &convID)
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	p, err := s.authorize(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	at, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if !p.Blocked {
		s.events.Publish(p.PeerID, realtime.Event{Type: "conversation.read", Data: map[string]any{
			"conversationId": conversationID, "userId": userID, "readAt": at,
		}})
	}
	return nil
}

// Clear hides the existing history for the caller only; the other participant keeps theirs.
func (s *Service) Clear(ctx context.Context, userID, conversationID string) error {
	if _, err := s.authorize(ctx, conversationID, userID); err != nil {
		return err
	}
	return s.repo.Clear(ctx, conversationID, userID)
}

func (s *Service) UnreadTotal(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadTotal(ctx, userID)
}
