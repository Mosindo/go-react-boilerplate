package discovery

import (
	"context"
	"time"

	"example.com/api/internal/features/profiles"
)

// ProfileReader is the subset of the profiles service discovery depends on.
type ProfileReader interface {
	MyProfile(ctx context.Context, userID string) (profiles.MyProfile, error)
	ViewerLocation(ctx context.Context, userID string) (lat, lng float64, ok bool, err error)
	PublicProfiles(ctx context.Context, viewerID string, userIDs []string) ([]profiles.PublicProfile, error)
}

type Service struct {
	repo     Repository
	profiles ProfileReader
	ranker   Ranker
	now      func() time.Time
}

func NewService(repo Repository, profileReader ProfileReader, ranker Ranker) *Service {
	return &Service{repo: repo, profiles: profileReader, ranker: ranker, now: time.Now}
}

func (s *Service) Feed(ctx context.Context, viewerID string, limit int) (FeedResponse, error) {
	if limit <= 0 || limit > MaxLimit {
		limit = DefaultLimit
	}
	me, err := s.profiles.MyProfile(ctx, viewerID)
	if err != nil {
		return FeedResponse{}, err
	}
	if !me.IsComplete {
		return FeedResponse{}, profiles.ErrProfileIncomplete
	}
	lat, lng, ok, err := s.profiles.ViewerLocation(ctx, viewerID)
	if err != nil {
		return FeedResponse{}, err
	}
	if !ok {
		return FeedResponse{}, profiles.ErrProfileIncomplete
	}
	_ = s.repo.TouchActivity(ctx, viewerID)

	pool := limit * poolFactor
	if pool > maxPool {
		pool = maxPool
	}
	box := BoundingBoxAround(lat, lng, me.Preferences.MaxDistanceKm)
	candidates, err := s.repo.Candidates(ctx, viewerID, box, pool)
	if err != nil {
		return FeedResponse{}, err
	}
	ranked := s.ranker.Rank(viewerID, candidates, me.Preferences.MaxDistanceKm, s.now())
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	ids := make([]string, 0, len(ranked))
	shared := make(map[string]int, len(ranked))
	for _, c := range ranked {
		ids = append(ids, c.UserID)
		shared[c.UserID] = c.SharedInterests
	}
	cards, err := s.profiles.PublicProfiles(ctx, viewerID, ids)
	if err != nil {
		return FeedResponse{}, err
	}
	out := make([]Card, 0, len(cards))
	for _, card := range cards {
		out = append(out, Card{PublicProfile: card, SharedInterests: shared[card.UserID]})
	}
	return FeedResponse{Profiles: out}, nil
}
