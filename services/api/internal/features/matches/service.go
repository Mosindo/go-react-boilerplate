package matches

import (
	"context"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

type ProfileReader interface {
	PublicProfiles(ctx context.Context, viewerID string, ids []string) (map[string]profiles.PublicProfile, error)
}

type Service struct {
	repo     *Repository
	profiles ProfileReader
	events   realtime.Publisher
}

func NewService(repo *Repository, profiles ProfileReader, events realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, events: events}
}

func (s *Service) List(ctx context.Context, userID string, limit int, cur *httpx.Cursor) (httpx.Page[Item], error) {
	rows, err := s.repo.list(ctx, userID, limit, cur)
	if err != nil {
		return httpx.Page[Item]{}, err
	}
	page := httpx.Paginate(rows, limit, func(r row) httpx.Cursor { return httpx.Cursor{At: r.CreatedAt, ID: r.MatchID} })

	ids := make([]string, len(page.Items))
	for i, rw := range page.Items {
		ids[i] = rw.OtherUserID
	}
	byID, err := s.profiles.PublicProfiles(ctx, userID, ids)
	if err != nil {
		return httpx.Page[Item]{}, err
	}
	items := make([]Item, 0, len(page.Items))
	for _, rw := range page.Items {
		p, ok := byID[rw.OtherUserID]
		if !ok {
			continue
		}
		items = append(items, Item{MatchID: rw.MatchID, ConversationID: rw.ConversationID, CreatedAt: rw.CreatedAt, Profile: p})
	}
	return httpx.Page[Item]{Items: items, NextCursor: page.NextCursor}, nil
}

// BuildItem assembles the match as seen by viewerID.
func (s *Service) BuildItem(ctx context.Context, viewerID, matchID, conversationID string, createdAt time.Time, otherID string) (Item, error) {
	byID, err := s.profiles.PublicProfiles(ctx, viewerID, []string{otherID})
	if err != nil {
		return Item{}, err
	}
	return Item{MatchID: matchID, ConversationID: conversationID, CreatedAt: createdAt, Profile: byID[otherID]}, nil
}

// Unmatch deletes the match and its conversation and tells both users' clients.
func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	rw, err := s.repo.remove(ctx, userID, matchID)
	if err != nil {
		return err
	}
	evt := realtime.Event{Type: "conversation_removed", Data: map[string]string{"conversationId": rw.ConversationID}}
	s.events.Publish(userID, evt)
	s.events.Publish(rw.OtherUserID, evt)
	return nil
}
