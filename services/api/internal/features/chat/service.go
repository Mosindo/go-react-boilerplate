package chat

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/validate"
)

var ErrInvalidMessage = errors.New("message must be between 1 and 2000 characters")

type CardLoader interface {
	Cards(ctx context.Context, viewerID string, ids []string) ([]profiles.Card, error)
}

type Service struct {
	repo   Repository
	cards  CardLoader
	events realtime.Publisher
}

func NewService(repo Repository, cards CardLoader, events realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, events: events}
}

func (s *Service) Conversations(ctx context.Context, userID string, limit int) ([]Conversation, error) {
	stored, err := s.repo.ListConversations(ctx, userID, clampPage(limit, 50))
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(stored))
	for i, c := range stored {
		ids[i] = c.OtherUserID
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.Card, len(cards))
	for _, c := range cards {
		byID[c.ID] = c
	}
	out := make([]Conversation, 0, len(stored))
	for _, c := range stored {
		card, ok := byID[c.OtherUserID]
		if !ok {
			continue
		}
		conv := Conversation{ID: c.ID, MatchID: c.MatchID, User: card, UnreadCount: c.UnreadCount, CreatedAt: c.CreatedAt, SortAt: c.CreatedAt}
		if c.LastBody != nil && c.LastSenderID != nil && c.LastMessageAt != nil {
			conv.LastMessage = &MessagePreview{Body: *c.LastBody, SenderID: *c.LastSenderID, CreatedAt: *c.LastMessageAt}
			conv.SortAt = *c.LastMessageAt
		}
		out = append(out, conv)
	}
	return out, nil
}

func (s *Service) Messages(ctx context.Context, userID, conversationID string, before *time.Time, limit int) (MessagesResponse, error) {
	if _, err := s.repo.Participant(ctx, userID, conversationID); err != nil {
		return MessagesResponse{}, err
	}
	limit = clampPage(limit, DefaultPage)
	msgs, err := s.repo.ListMessages(ctx, conversationID, before, limit+1)
	if err != nil {
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{}
	if len(msgs) > limit {
		msgs = msgs[:limit]
		next := msgs[len(msgs)-1].CreatedAt
		resp.NextBefore = &next
	}
	resp.Messages = msgs
	if resp.OtherLastReadAt, err = s.repo.OtherLastRead(ctx, conversationID, userID); err != nil {
		return MessagesResponse{}, err
	}
	return resp, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, body string) (Message, error) {
	clean, ok := validate.CleanText(body, MaxMessageRunes)
	if !ok || clean == "" {
		return Message{}, ErrInvalidMessage
	}
	other, err := s.repo.Participant(ctx, userID, conversationID)
	if err != nil {
		return Message{}, err
	}
	name := ""
	if cards, err := s.cards.Cards(ctx, other, []string{userID}); err == nil && len(cards) > 0 {
		name = cards[0].FirstName
	}
	msg, err := s.repo.InsertMessage(ctx, conversationID, userID, other, name, clean)
	if err != nil {
		return Message{}, err
	}
	s.events.Publish(other, "message", msg)
	s.events.Publish(userID, "message", msg) // keep the sender's other devices in sync
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) error {
	other, err := s.repo.Participant(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	if err := s.repo.MarkRead(ctx, conversationID, userID); err != nil {
		return err
	}
	s.events.Publish(other, "read", map[string]any{"conversationId": conversationID, "readAt": time.Now().UTC()})
	return nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	if _, err := s.repo.Participant(ctx, userID, conversationID); err != nil {
		return err
	}
	return s.repo.Hide(ctx, conversationID, userID)
}

func clampPage(limit, def int) int {
	if limit <= 0 {
		return def
	}
	if limit > MaxPage {
		return MaxPage
	}
	return limit
}
