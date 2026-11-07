package discovery

import (
	"context"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

const (
	DefaultBatch = 10
	MaxBatch     = 20
)

type CardLoader interface {
	Cards(ctx context.Context, viewerID string, ids []string) ([]profiles.Card, error)
}

type Service struct {
	repo  Repository
	cards CardLoader
	rt    realtime.Publisher
}

func NewService(repo Repository, cards CardLoader, rt realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, rt: rt}
}

func (s *Service) requireComplete(ctx context.Context, userID string) error {
	ok, err := s.repo.IsComplete(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrProfileIncomplete
	}
	return nil
}

// Feed returns the next batch of profiles to review.
func (s *Service) Feed(ctx context.Context, userID string, limit int) ([]profiles.Card, error) {
	if err := s.requireComplete(ctx, userID); err != nil {
		return nil, err
	}
	ids, err := s.repo.Candidates(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	return s.cards.Cards(ctx, userID, ids)
}

func (s *Service) LikesReceived(ctx context.Context, userID string, limit, offset int) ([]profiles.Card, error) {
	if err := s.requireComplete(ctx, userID); err != nil {
		return nil, err
	}
	ids, err := s.repo.LikesReceived(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.cards.Cards(ctx, userID, ids)
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResponse, error) {
	if err := s.requireComplete(ctx, userID); err != nil {
		return SwipeResponse{}, err
	}
	if userID == req.UserID {
		return SwipeResponse{}, ErrTargetUnavailable
	}
	m, err := s.repo.Swipe(ctx, userID, req.UserID, req.Action)
	if err != nil {
		return SwipeResponse{}, err
	}
	resp := SwipeResponse{Action: req.Action}
	if m == nil {
		if req.Action == "like" {
			s.rt.Publish(req.UserID, realtime.Event{Type: "notification", Data: map[string]string{"kind": "like"}})
		}
		return resp, nil
	}
	resp.Matched = true
	views, err := s.matchViews(ctx, userID, []storedMatch{*m})
	if err != nil {
		return SwipeResponse{}, err
	}
	if len(views) == 1 {
		resp.Match = &views[0]
	}
	if other, err := s.matchViews(ctx, req.UserID, []storedMatch{{ID: m.ID, ConversationID: m.ConversationID, OtherUserID: userID, CreatedAt: m.CreatedAt}}); err == nil && len(other) == 1 {
		s.rt.Publish(req.UserID, realtime.Event{Type: "match", Data: other[0]})
	}
	return resp, nil
}

func (s *Service) Matches(ctx context.Context, userID string, limit, offset int) ([]MatchView, error) {
	list, err := s.repo.Matches(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.matchViews(ctx, userID, list)
}

func (s *Service) matchViews(ctx context.Context, viewerID string, list []storedMatch) ([]MatchView, error) {
	ids := make([]string, 0, len(list))
	for _, m := range list {
		ids = append(ids, m.OtherUserID)
	}
	cards, err := s.cards.Cards(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.Card, len(cards))
	for _, c := range cards {
		byID[c.UserID] = c
	}
	out := make([]MatchView, 0, len(list))
	for _, m := range list {
		if card, ok := byID[m.OtherUserID]; ok {
			out = append(out, MatchView{ID: m.ID, ConversationID: m.ConversationID, CreatedAt: m.CreatedAt, User: card})
		}
	}
	return out, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.rt.Publish(other, realtime.Event{Type: "unmatch", Data: map[string]string{"matchId": matchID}})
	return nil
}
