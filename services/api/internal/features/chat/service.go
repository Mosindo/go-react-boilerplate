package chat

import (
	"context"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	rt   realtime.Publisher
}

func NewService(repo Repository, rt realtime.Publisher) *Service { return &Service{repo: repo, rt: rt} }

func (s *Service) Conversations(ctx context.Context, userID string, limit int, cur *httpx.Cursor) ([]Conversation, error) {
	return s.repo.ListConversations(ctx, userID, limit, cur)
}

func (s *Service) Messages(ctx context.Context, userID, convID string, limit int, cur *httpx.Cursor) (MessagePage, error) {
	return s.repo.ListMessages(ctx, userID, convID, limit, cur)
}

func (s *Service) Send(ctx context.Context, userID, convID, rawBody string) (Message, error) {
	body, err := CleanBody(rawBody)
	if err != nil {
		return Message{}, err
	}
	m, otherID, err := s.repo.Send(ctx, userID, convID, body)
	if err != nil {
		return Message{}, err
	}
	ev := realtime.Event{Type: "message", Payload: m}
	s.rt.Publish(otherID, ev)
	s.rt.Publish(userID, ev) // synchronise les autres appareils de l'expediteur
	return m, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, convID string) error {
	otherID, readAt, err := s.repo.MarkRead(ctx, userID, convID)
	if err != nil {
		return err
	}
	s.rt.Publish(otherID, realtime.Event{Type: "read", Payload: ReadEvent{ConversationID: convID, ReadAt: readAt}})
	return nil
}

func (s *Service) Clear(ctx context.Context, userID, convID string) error {
	return s.repo.Clear(ctx, userID, convID)
}
