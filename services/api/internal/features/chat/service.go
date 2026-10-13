package chat

import (
	"context"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

type ProfileSource interface {
	PublicByIDs(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
}

type Service struct {
	repo     Repository
	profiles ProfileSource
	events   realtime.Publisher
}

func NewService(repo Repository, source ProfileSource, events realtime.Publisher) *Service {
	if events == nil {
		events = realtime.Discard{}
	}
	return &Service{repo: repo, profiles: source, events: events}
}

func clamp(v, def, max int) int {
	if v < 1 {
		return def
	}
	if v > max {
		return max
	}
	return v
}

func (s *Service) Conversations(ctx context.Context, userID string, before *time.Time, limit int) (ConversationsResponse, error) {
	limit = clamp(limit, DefaultListLimit, MaxListLimit)
	stored, err := s.repo.ListConversations(ctx, userID, before, limit)
	if err != nil {
		return ConversationsResponse{}, err
	}
	ids := make([]string, len(stored))
	for i, c := range stored {
		ids[i] = c.OtherUserID
	}
	people, err := s.profiles.PublicByIDs(ctx, userID, ids)
	if err != nil {
		return ConversationsResponse{}, err
	}
	byID := make(map[string]profiles.PublicProfile, len(people))
	for _, p := range people {
		byID[p.ID] = p
	}
	resp := ConversationsResponse{Conversations: make([]Conversation, 0, len(stored))}
	for _, c := range stored {
		p, ok := byID[c.OtherUserID]
		if !ok {
			continue
		}
		resp.Conversations = append(resp.Conversations, Conversation{
			ID: c.ID, MatchID: c.MatchID, Profile: p, LastMessage: c.LastMessage,
			UnreadCount: c.UnreadCount, UpdatedAt: c.LastMessageAt,
		})
	}
	if len(stored) == limit {
		last := stored[len(stored)-1].LastMessageAt
		resp.NextBefore = &last
	}
	return resp, nil
}

func (s *Service) Messages(ctx context.Context, userID, conversationID string, before *int64, limit int) (MessagesResponse, error) {
	if _, err := s.repo.Access(ctx, userID, conversationID); err != nil {
		return MessagesResponse{}, err
	}
	limit = clamp(limit, DefaultMessageLimit, MaxMessageLimit)
	msgs, err := s.repo.ListMessages(ctx, userID, conversationID, before, limit)
	if err != nil {
		return MessagesResponse{}, err
	}
	resp := MessagesResponse{Messages: msgs}
	if len(msgs) == limit {
		oldest := msgs[0].ID
		resp.NextBefore = &oldest
	}
	return resp, nil
}

func (s *Service) Send(ctx context.Context, userID, conversationID, rawBody string) (Message, error) {
	body, err := NormalizeMessage(rawBody)
	if err != nil {
		return Message{}, err
	}
	msg, recipient, err := s.repo.InsertMessage(ctx, userID, conversationID, body)
	if err != nil {
		return Message{}, err
	}
	s.events.Publish(recipient, realtime.Event{Type: "message", Data: msg})
	s.events.Publish(recipient, realtime.Event{Type: "notification"})
	s.events.Publish(userID, realtime.Event{Type: "message", Data: msg})
	return msg, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, conversationID string) (ReadResult, error) {
	other, err := s.repo.Access(ctx, userID, conversationID)
	if err != nil {
		return ReadResult{}, err
	}
	readAt, err := s.repo.MarkRead(ctx, userID, conversationID)
	if err != nil {
		return ReadResult{}, err
	}
	s.events.Publish(other, realtime.Event{Type: "read", Data: map[string]any{
		"conversationId": conversationID, "readAt": readAt,
	}})
	return ReadResult{ReadAt: readAt}, nil
}

func (s *Service) Hide(ctx context.Context, userID, conversationID string) error {
	return s.repo.Hide(ctx, userID, conversationID)
}
