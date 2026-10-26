package matching

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrProfileIncomplete = errors.New("complete your profile (preferences, location and a photo) to start discovering")
	ErrUnavailable       = errors.New("this profile is not available")
	ErrNotFound          = errors.New("match not found")
)

// Service holds the matching rules. Candidate selection is isolated in the
// repository's eligibility predicate plus a ranking ORDER BY, which is the
// single place to evolve the recommendation algorithm.
type Service struct {
	repo   Repository
	loader *profiles.Loader
	db     profiles.Querier
	pub    realtime.Publisher
}

func NewService(repo Repository, loader *profiles.Loader, db profiles.Querier, pub realtime.Publisher) *Service {
	return &Service{repo: repo, loader: loader, db: db, pub: pub}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) (DiscoverResponse, error) {
	if limit <= 0 {
		limit = DefaultBatch
	}
	if limit > MaxBatch {
		limit = MaxBatch
	}
	ready, err := s.repo.ViewerReady(ctx, userID)
	if err != nil {
		return DiscoverResponse{}, err
	}
	if !ready {
		return DiscoverResponse{}, ErrProfileIncomplete
	}
	_ = s.repo.TouchActive(ctx, userID)

	ids, err := s.repo.Candidates(ctx, userID, limit)
	if err != nil {
		return DiscoverResponse{}, err
	}
	views, err := s.loader.Load(ctx, s.db, userID, ids)
	if err != nil {
		return DiscoverResponse{}, err
	}
	out := DiscoverResponse{Profiles: make([]profiles.PublicProfile, 0, len(ids))}
	for _, id := range ids {
		if p, ok := views[id]; ok {
			out.Profiles = append(out.Profiles, p)
		}
	}
	return out, nil
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResponse, error) {
	if req.UserID == userID {
		return SwipeResponse{}, ErrUnavailable
	}
	ready, err := s.repo.ViewerReady(ctx, userID)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !ready {
		return SwipeResponse{}, ErrProfileIncomplete
	}
	outcome, err := s.repo.Swipe(ctx, userID, req.UserID, req.Action)
	if err != nil {
		if errors.Is(err, ErrRepositoryUnavailable) {
			return SwipeResponse{}, ErrUnavailable
		}
		return SwipeResponse{}, err
	}

	resp := SwipeResponse{AlreadySwiped: outcome.AlreadySwiped}
	if outcome.MatchID == "" {
		return resp, nil
	}
	views, err := s.loader.Load(ctx, s.db, userID, []string{req.UserID})
	if err != nil {
		return SwipeResponse{}, err
	}
	resp.Matched = !outcome.AlreadySwiped
	resp.Match = &MatchView{ID: outcome.MatchID, CreatedAt: outcome.MatchedAt, Profile: views[req.UserID]}

	if resp.Matched {
		// Tell the other person in real time, with the swiper's profile as seen by them.
		if theirView, err := s.loader.Load(ctx, s.db, req.UserID, []string{userID}); err == nil {
			s.pub.Publish(req.UserID, realtime.Event{Type: "match", Data: MatchView{
				ID: outcome.MatchID, CreatedAt: outcome.MatchedAt, Profile: theirView[userID],
			}})
		}
	}
	return resp, nil
}

func (s *Service) GetMatch(ctx context.Context, userID, matchID string) (MatchView, error) {
	other, created, err := s.repo.GetMatch(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return MatchView{}, ErrNotFound
		}
		return MatchView{}, err
	}
	views, err := s.loader.Load(ctx, s.db, userID, []string{other})
	if err != nil {
		return MatchView{}, err
	}
	return MatchView{ID: matchID, CreatedAt: created, Profile: views[other]}, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrNotFound
		}
		return err
	}
	s.pub.Publish(other, realtime.Event{Type: "match_ended", Data: map[string]string{"matchId": matchID}})
	return nil
}
