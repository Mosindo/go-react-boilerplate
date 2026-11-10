package discovery

import (
	"context"
	"errors"
	"math/rand/v2"
	"strings"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/httpx"
)

var (
	ErrProfileIncomplete = errors.New("complete your profile (photo, preferences and location) to discover people")
	ErrSuspended         = errors.New("your account is suspended")
	ErrNotAvailable      = errors.New("this profile is no longer available")
	ErrAlreadySwiped     = errors.New("you already answered this profile")
	ErrSelfSwipe         = errors.New("you cannot swipe yourself")
	ErrInvalidExclude    = errors.New("invalid exclude list")
)

// MatchListener is implemented by the matches module.
type MatchListener interface {
	MatchCreated(ctx context.Context, matchID, userA, userB string)
}

type Service struct {
	repo     Repository
	profiles *profiles.Service
	matches  MatchListener
	now      func() time.Time
	jitter   func() float64
}

func NewService(repo Repository, profilesSvc *profiles.Service, matches MatchListener) *Service {
	return &Service{repo: repo, profiles: profilesSvc, matches: matches, now: time.Now, jitter: rand.Float64}
}

// ParseExclude validates the ids the client already holds in its deck.
func ParseExclude(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) > MaxExcluded {
		return nil, ErrInvalidExclude
	}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if !httpx.IsUUID(p) {
			return nil, ErrInvalidExclude
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *Service) Deck(ctx context.Context, userID string, exclude []string, limit int) ([]profiles.Card, error) {
	v, err := s.repo.GetViewer(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepoViewerIncomplete) {
			return nil, ErrProfileIncomplete
		}
		return nil, err
	}
	if v.Suspended {
		return nil, ErrSuspended
	}
	minLat, maxLat, minLon, maxLon := geo.BoundingBox(v.Latitude, v.Longitude, float64(v.MaxDistanceKm))
	cands, err := s.repo.Candidates(ctx, v, minLat, maxLat, minLon, maxLon, exclude, CandidatePool)
	if err != nil {
		return nil, err
	}
	for i := range cands {
		cands[i].DistanceKm = geo.DistanceKm(v.Latitude, v.Longitude, cands[i].Latitude, cands[i].Longitude)
	}
	ranked := Rank(cands, v.MaxDistanceKm, s.now(), s.jitter, limit)
	ids := make([]string, 0, len(ranked))
	for _, c := range ranked {
		ids = append(ids, c.UserID)
	}
	return s.profiles.Cards(ctx, userID, ids)
}

func (s *Service) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, error) {
	targetID = strings.ToLower(targetID)
	if userID == targetID {
		return SwipeResult{}, ErrSelfSwipe
	}
	if action != ActionLike && action != ActionPass {
		return SwipeResult{}, errors.New("invalid action")
	}
	v, err := s.repo.GetViewer(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepoViewerIncomplete) {
			return SwipeResult{}, ErrProfileIncomplete
		}
		return SwipeResult{}, err
	}
	if v.Suspended {
		return SwipeResult{}, ErrSuspended
	}
	res, err := s.repo.Swipe(ctx, userID, targetID, action)
	if err != nil {
		switch {
		case errors.Is(err, ErrRepoNotEligible):
			return SwipeResult{}, ErrNotAvailable
		case errors.Is(err, ErrRepoAlreadySwiped):
			return SwipeResult{}, ErrAlreadySwiped
		}
		return SwipeResult{}, err
	}
	if res.Matched {
		s.matches.MatchCreated(ctx, res.MatchID, userID, targetID)
	}
	return res, nil
}
