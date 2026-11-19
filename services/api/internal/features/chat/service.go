package chat

import (
	"context"
	"log"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

type CardLoader interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Card, error)
}

type MessageNotifier interface {
	NotifyMessage(ctx context.Context, userID, actorID, conversationID string) error
	MarkConversationRead(ctx context.Context, userID, conversationID string) error
}

type Service struct {
	repo      Repository
	cards     CardLoader
	notifier  MessageNotifier
	publisher realtime.Publisher
}

func NewService(repo Repository, cards CardLoader, notifier MessageNotifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, notifier: notifier, publisher: publisher}
}

// NormalizeBody trims a message and strips control characters (newlines and tabs are kept).
func NormalizeBody(body string) (string, error) {
	var b strings.Builder
	for _, r := range body {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			continue
		}
		b.WriteRune(r)
	}
	out := strings.TrimSpace(b.String())
	if n := utf8.RuneCountInString(out); n == 0 || n > MaxMessageLength {
		return "", ErrInvalidMessage
	}
	return out, nil
}

func (s *Service) List(ctx context.Context, userID, cursor string, limit int) (ConversationsResponse, error) {
	var before *time.Time
	if cursor != "" {
		t, err := s.repo.ConversationCursor(ctx, userID, cursor)
		if err != nil {
			return ConversationsResponse{}, err
		}
		before = &t
	}
	stored, err := s.repo.ListConversations(ctx, userID, before, cursor, limit+1)
	if err != nil {
		return ConversationsResponse{}, err
	}
	var next *string
	if len(stored) > limit {
		stored = stored[:limit]
		id := stored[limit-1].ConversationID
		next = &id
	}
	items, err := s.decorate(ctx, userID, stored)
	if err != nil {
		return ConversationsResponse{}, err
	}
	return ConversationsResponse{Conversations: items, NextCursor: next}, nil
}

// Get returns a single conversation summary (used when opening a thread from a notification).
func (s *Service) Get(ctx context.Context, userID, conversationID string) (Conversation, error) {
	m, err := s.repo.Membership(ctx, conversationID, userID)
	if err != nil {
		return Conversation{}, err
	}
	if !m.Active {
		return Conversation{}, ErrNotFound
	}
	items, err := s.decorate(ctx, userID, []StoredSummary{{
		ConversationID:    m.ConversationID,
		MatchID:           m.MatchID,
		PartnerID:         m.PartnerID,
		MatchedAt:         m.MatchedAt,
		PartnerLastReadAt: m.PartnerLastReadAt,
	}})
	if err != nil {
		return Conversation{}, err
	}
	return items[0], nil
}

func (s *Service) decorate(ctx context.Context, userID string, stored []StoredSummary) ([]Conversation, error) {
	ids := make([]string, 0, len(stored))
	for _, c := range stored {
		ids = append(ids, c.PartnerID)
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Conversation, 0, len(stored))
	for _, c := range stored {
		card := cards[c.PartnerID]
		partner := Partner{UserID: c.PartnerID, FirstName: card.FirstName, Age: card.Age}
		if len(card.Photos) > 0 {
			partner.PhotoURL = &card.Photos[0].URL
		}
		conv := Conversation{ID: c.ConversationID, MatchID: c.MatchID, MatchedAt: c.MatchedAt, Partner: partner, UnreadCount: c.UnreadCount}
		if c.LastMessage != nil {
			msg := toMessage(*c.LastMessage, userID, c.PartnerLastReadAt)
			conv.LastMessage = &msg
		}
		out = append(out, conv)
	}
	return out, nil
}

func (s *Service) Messages(ctx context.Context, userID, conversationID, cursor string, limit int) (MessagesResponse, error) {
	m, err := s.repo.Membership(ctx, conversationID, userID)
	if err != nil {
		return MessagesResponse{}, err
	}
	if !m.Active {
		return MessagesResponse{}, ErrNotFound
	}
	var before *time.Time
	if cursor != "" {
		t, err := s.repo.MessageCursor(ctx, conversationID, cursor)
		if err != nil {
			return MessagesResponse{}, err
		}
		before = &t
	}
	stored, err := s.repo.ListMessages(ctx, conversationID, m.ClearedAt, before, cursor, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	var next *string
	if len(stored) > limit {
		stored = stored[:limit]
		id := stored[limit-1].ID
		next = &id
	}
	out := make([]Message, 0, len(stored))
	for _, msg := range stored {
		out = append(out, toMessage(msg, userID, m.PartnerLastReadAt))
	}
	return MessagesResponse{Messages: out, NextCursor: next, PartnerLastReadAt: m.PartnerLastReadAt}, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	clean, err := NormalizeBody(body)
	if err != nil {
		return Message{}, err
	}
	m, err := s.repo.Membership(ctx, conversationID, userID)
	if err != nil {
		return Message{}, err
	}
	if !m.Active {
		return Message{}, ErrClosed
	}
	stored, err := s.repo.InsertMessage(ctx, conversationID, userID, clean)
	if err != nil {
		return Message{}, err
	}

	// The recipient's view: not mine, unread.
	s.publisher.Publish(m.PartnerID, realtime.Event{Type: "message.new", Payload: toMessage(stored, m.PartnerID, nil)})
	mine := toMessage(stored, userID, m.PartnerLastReadAt)
	// Other devices of the sender stay in sync.
	s.publisher.Publish(userID, realtime.Event{Type: "message.new", Payload: mine})
	if err := s.notifier.NotifyMessage(ctx, m.PartnerID, userID, conversationID); err != nil {
		log.Printf(`{"event":"message_notification_failed","error":%q}`, err.Error())
	}
	return mine, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	m, err := s.repo.Membership(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	readAt, err := s.repo.MarkRead(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if err := s.notifier.MarkConversationRead(ctx, userID, conversationID); err != nil {
		log.Printf(`{"event":"notification_read_failed","error":%q}`, err.Error())
	}
	if m.Active {
		s.publisher.Publish(m.PartnerID, realtime.Event{
			Type:    "conversation.read",
			Payload: map[string]any{"conversationId": conversationID, "readAt": readAt},
		})
	}
	return nil
}

// Clear hides the conversation history for the requesting user only.
func (s *Service) Clear(ctx context.Context, userID, conversationID string) error {
	if _, err := s.repo.Membership(ctx, conversationID, userID); err != nil {
		return err
	}
	if err := s.repo.Clear(ctx, conversationID, userID); err != nil {
		return err
	}
	return s.notifier.MarkConversationRead(ctx, userID, conversationID)
}

func toMessage(m StoredMessage, viewerID string, partnerLastReadAt *time.Time) Message {
	mine := m.SenderID == viewerID
	read := false
	if mine && partnerLastReadAt != nil {
		read = !m.CreatedAt.After(*partnerLastReadAt)
	}
	return Message{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Body:           m.Body,
		CreatedAt:      m.CreatedAt,
		IsMine:         mine,
		IsRead:         read,
	}
}
