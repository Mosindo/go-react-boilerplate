package discovery

import (
	"context"
	"time"

	"example.com/api/internal/features/profiles"
	apperr "example.com/api/internal/platform/errors"
)

const candidatePoolSize = 100

var ErrProfileIncomplete = apperr.Conflict("profile_incomplete", "complete your profile before discovering people")

type CardProvider interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) ([]profiles.PublicProfile, error)
}

type Service struct {
	repo   Repository
	cards  CardProvider
	scorer Scorer
	now    func() time.Time
}

func NewService(repo Repository, cards CardProvider, scorer Scorer) *Service {
	if scorer == nil {
		scorer = DefaultScorer{}
	}
	return &Service{repo: repo, cards: cards, scorer: scorer, now: time.Now}
}

// Next returns the next profiles to show. Swiped profiles are excluded by the
// query, so calling Next again naturally yields the following batch.
func (s *Service) Next(ctx context.Context, viewerID string, limit int) ([]profiles.PublicProfile, error) {
	complete, err := s.repo.ViewerComplete(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if !complete {
		return nil, ErrProfileIncomplete
	}

	candidates, err := s.repo.Candidates(ctx, viewerID, candidatePoolSize)
	if err != nil {
		return nil, err
	}
	ranked := Rank(candidates, s.scorer, s.now())
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	ids := make([]string, len(ranked))
	for i, c := range ranked {
		ids[i] = c.UserID
	}
	return s.cards.Cards(ctx, viewerID, ids)
}
