package notifications

import (
	"context"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

type CardLoader interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Card, error)
}

type Service struct {
	repo      Repository
	cards     CardLoader
	publisher realtime.Publisher
}

func NewService(repo Repository, cards CardLoader, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, publisher: publisher}
}

func (s *Service) NotifyMatch(ctx context.Context, userID, actorID, conversationID string) error {
	return s.create(ctx, userID, TypeMatch, actorID, conversationID)
}

func (s *Service) NotifyMessage(ctx context.Context, userID, actorID, conversationID string) error {
	return s.create(ctx, userID, TypeMessage, actorID, conversationID)
}

func (s *Service) create(ctx context.Context, userID, kind, actorID, conversationID string) error {
	if _, err := s.repo.Create(ctx, userID, kind, &actorID, &conversationID); err != nil {
		return err
	}
	return s.publishCount(ctx, userID)
}

func (s *Service) publishCount(ctx context.Context, userID string) error {
	count, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return err
	}
	s.publisher.Publish(userID, realtime.Event{Type: "notifications.unread", Payload: map[string]int{"unreadCount": count}})
	return nil
}

func (s *Service) List(ctx context.Context, userID, cursor string, limit int) (ListResponse, error) {
	var before *time.Time
	if cursor != "" {
		t, err := s.repo.CursorOf(ctx, userID, cursor)
		if err != nil {
			return ListResponse{}, err
		}
		before = &t
	}
	stored, err := s.repo.List(ctx, userID, before, cursor, limit+1)
	if err != nil {
		return ListResponse{}, err
	}
	var next *string
	if len(stored) > limit {
		stored = stored[:limit]
		id := stored[limit-1].ID
		next = &id
	}

	actorIDs := make([]string, 0, len(stored))
	seen := map[string]bool{}
	for _, n := range stored {
		if n.ActorID != nil && !seen[*n.ActorID] {
			seen[*n.ActorID] = true
			actorIDs = append(actorIDs, *n.ActorID)
		}
	}
	cards, err := s.cards.Cards(ctx, userID, actorIDs)
	if err != nil {
		return ListResponse{}, err
	}

	items := make([]Notification, 0, len(stored))
	for _, n := range stored {
		item := Notification{ID: n.ID, Type: n.Type, ConversationID: n.ConversationID, ReadAt: n.ReadAt, CreatedAt: n.CreatedAt}
		if n.ActorID != nil {
			if card, ok := cards[*n.ActorID]; ok {
				actor := Actor{UserID: card.UserID, FirstName: card.FirstName}
				if len(card.Photos) > 0 {
					actor.PhotoURL = &card.Photos[0].URL
				}
				item.Actor = &actor
			}
		}
		items = append(items, item)
	}

	unread, err := s.repo.UnreadCount(ctx, userID)
	if err != nil {
		return ListResponse{}, err
	}
	return ListResponse{Notifications: items, NextCursor: next, UnreadCount: unread}, nil
}

func (s *Service) UnreadCount(ctx context.Context, userID string) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) MarkRead(ctx context.Context, userID, id string) error {
	if err := s.repo.MarkRead(ctx, userID, id); err != nil {
		return err
	}
	return s.publishCount(ctx, userID)
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) error {
	if err := s.repo.MarkAllRead(ctx, userID); err != nil {
		return err
	}
	return s.publishCount(ctx, userID)
}

// MarkConversationRead clears message notifications once a conversation was opened.
func (s *Service) MarkConversationRead(ctx context.Context, userID, conversationID string) error {
	if err := s.repo.MarkConversationRead(ctx, userID, conversationID); err != nil {
		return err
	}
	return s.publishCount(ctx, userID)
}
