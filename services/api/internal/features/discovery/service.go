package discovery

import (
	"context"
	"time"

	"example.com/api/internal/features/profiles"
)

const (
	DefaultLimit = 10
	MaxLimit     = 30
	poolSize     = 200
)

type CardLoader interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) ([]profiles.Card, error)
}

type Service struct {
	repo        Repository
	cards       CardLoader
	ranker      Ranker
	includeDemo bool
	now         func() time.Time
}

func NewService(repo Repository, cards CardLoader, ranker Ranker, includeDemo bool) *Service {
	return &Service{repo: repo, cards: cards, ranker: ranker, includeDemo: includeDemo, now: time.Now}
}

func (s *Service) Deck(ctx context.Context, viewerID string, limit int) (Response, error) {
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}
	state, err := s.repo.ViewerState(ctx, viewerID)
	if err != nil {
		return Response{}, err
	}
	if !state.HasProfile || !state.HasPreferences || !state.HasPhotos {
		return Response{}, ErrViewerIncomplete
	}
	_ = s.repo.TouchActivity(ctx, viewerID)

	candidates, err := s.repo.Candidates(ctx, viewerID, poolSize, s.includeDemo)
	if err != nil {
		return Response{}, err
	}
	ranked := s.ranker.Rank(viewerID, candidates, s.now())
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	ids := make([]string, len(ranked))
	for i, c := range ranked {
		ids[i] = c.UserID
	}
	cards, err := s.cards.Cards(ctx, viewerID, ids)
	if err != nil {
		return Response{}, err
	}
	resp := Response{Profiles: cards}
	if len(cards) == 0 {
		resp.Reason = "no_more_profiles"
		if !state.IsDiscoverable {
			resp.Reason = "no_more_profiles_paused"
		}
	}
	return resp, nil
}
