package discovery

import (
	"context"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/geo"
)

const (
	DefaultFeedSize = 10
	MaxFeedSize     = 20
	poolSize        = 100
	maxExclude      = 50
)

type Service struct {
	repo     Repository
	ranker   Ranker
	profiles *profiles.Service
	now      func() time.Time
}

func NewService(repo Repository, ranker Ranker, profilesService *profiles.Service) *Service {
	if ranker == nil {
		ranker = NewScoreRanker()
	}
	return &Service{repo: repo, ranker: ranker, profiles: profilesService, now: time.Now}
}

// Feed returns the next profiles to show. excludeIDs lets the client skip
// cards it already holds locally but has not swiped yet.
func (s *Service) Feed(ctx context.Context, userID string, limit int, excludeIDs []string) ([]FeedItem, error) {
	if limit <= 0 {
		limit = DefaultFeedSize
	}
	if limit > MaxFeedSize {
		limit = MaxFeedSize
	}
	if len(excludeIDs) > maxExclude {
		excludeIDs = excludeIDs[:maxExclude]
	}

	viewer, err := s.repo.LoadViewer(ctx, userID)
	if err != nil {
		return nil, err
	}
	s.profiles.TouchActive(ctx, userID)

	q := CandidateQuery{Viewer: viewer, ExcludeIDs: excludeIDs, PoolSize: poolSize}
	if viewer.Latitude != nil {
		q.LatDelta, q.LngDelta = geo.BoundingBox(*viewer.Latitude, float64(viewer.MaxDistanceKm))
	}
	candidates, err := s.repo.ListCandidates(ctx, q)
	if err != nil {
		return nil, err
	}
	ranked := s.ranker.Rank(viewer, candidates, s.now())
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	ids := make([]string, 0, len(ranked))
	meta := make(map[string]Candidate, len(ranked))
	for _, c := range ranked {
		ids = append(ids, c.UserID)
		meta[c.UserID] = c
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	items := make([]FeedItem, 0, len(cards))
	for _, card := range cards {
		m := meta[card.UserID]
		items = append(items, FeedItem{CardResponse: card, LikedYou: m.LikedViewer, SharedInterests: m.SharedInterests})
	}
	return items, nil
}
