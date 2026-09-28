package discovery

import (
	"context"
	"errors"
	"math"
	"time"

	"example.com/api/internal/features/matches"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/events"
	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/validate"
)

type Service struct {
	repo   Repository
	ranker Ranker
	pub    events.Publisher
	now    func() time.Time
}

func NewService(repo Repository, ranker Ranker, pub events.Publisher) *Service {
	if ranker == nil {
		ranker = DefaultRanker{}
	}
	if pub == nil {
		pub = events.Nop{}
	}
	return &Service{repo: repo, ranker: ranker, pub: pub, now: time.Now}
}

func (s *Service) completeViewer(ctx context.Context, userID string) (Viewer, error) {
	v, err := s.repo.GetViewer(ctx, userID)
	if err != nil {
		return Viewer{}, err
	}
	if !v.Complete() {
		return Viewer{}, httpx.ProfileIncomplete()
	}
	return v, nil
}

// Discover returns up to `limit` candidates. The SQL filter yields a pool of
// the most recently active eligible profiles (4x limit, at least 40); the
// Ranker orders that pool and the head is returned.
func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]Candidate, error) {
	v, err := s.completeViewer(ctx, userID)
	if err != nil {
		return nil, err
	}
	s.repo.TouchActive(ctx, userID)

	now := s.now().UTC()
	pool := limit * 4
	if pool < 40 {
		pool = 40
	}
	rows, err := s.repo.Candidates(ctx, CandidateQuery{
		Viewer:    v,
		ViewerAge: validate.AgeOn(v.BirthDate, now),
		Today:     time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		PoolSize:  pool,
	})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []Candidate{}, nil
	}

	byID := make(map[string]CandidateRow, len(rows))
	items := make([]RankItem, len(rows))
	for i, r := range rows {
		byID[r.UserID] = r
		dist := math.NaN()
		if r.DistanceKm != nil {
			dist = *r.DistanceKm
		}
		items[i] = RankItem{UserID: r.UserID, DistanceKm: dist, SharedInterests: r.SharedInterests, LastActiveAt: r.LastActiveAt}
	}
	ranked := s.ranker.Rank(now, items)
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	ids := make([]string, len(ranked))
	for i, it := range ranked {
		ids[i] = it.UserID
	}
	return s.assemble(ctx, now, ids, byID)
}

// assemble batches interests and photos for the page (2 queries total).
func (s *Service) assemble(ctx context.Context, now time.Time, ids []string, byID map[string]CandidateRow) ([]Candidate, error) {
	interests, err := s.repo.Interests(ctx, ids)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.Photos(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Candidate, 0, len(ids))
	for _, id := range ids {
		out = append(out, toCandidate(byID[id], now, interests[id], photos[id]))
	}
	return out, nil
}

func toCandidate(r CandidateRow, now time.Time, interests []profiles.Interest, photos []profiles.Photo) Candidate {
	c := Candidate{
		UserID:              r.UserID,
		FirstName:           r.FirstName,
		Age:                 validate.AgeOn(r.BirthDate, now),
		Bio:                 r.Bio,
		LocationLabel:       r.LocationLabel,
		Interests:           interests,
		SharedInterestCount: r.SharedInterests,
		Photos:              photos,
	}
	if c.Interests == nil {
		c.Interests = []profiles.Interest{}
	}
	if c.Photos == nil {
		c.Photos = []profiles.Photo{}
	}
	if r.ShowDistance && r.DistanceKm != nil {
		b := geo.BucketKm(*r.DistanceKm)
		c.DistanceKm = &b
	}
	return c
}

func (s *Service) Profile(ctx context.Context, viewerID, targetID string) (Candidate, error) {
	id, ok := httpx.NormalizeUUID(targetID)
	if !ok {
		return Candidate{}, httpx.NotFound("user not found")
	}
	v, err := s.completeViewer(ctx, viewerID)
	if err != nil {
		return Candidate{}, err
	}
	row, err := s.repo.ProfileFor(ctx, v, id)
	if err != nil {
		return Candidate{}, err
	}
	if row == nil {
		return Candidate{}, httpx.NotFound("user not found")
	}
	now := s.now().UTC()
	interests, err := s.repo.Interests(ctx, []string{id})
	if err != nil {
		return Candidate{}, err
	}
	photos, err := s.repo.Photos(ctx, []string{id})
	if err != nil {
		return Candidate{}, err
	}
	return toCandidate(*row, now, interests[id], photos[id]), nil
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResponse, error) {
	target, ok := httpx.NormalizeUUID(req.TargetUserID)
	if !ok {
		return SwipeResponse{}, httpx.BadRequest("targetUserId must be a user id")
	}
	if req.Action != ActionLike && req.Action != ActionPass {
		return SwipeResponse{}, httpx.BadRequest("action must be 'like' or 'pass'")
	}
	if target == userID {
		return SwipeResponse{}, httpx.BadRequest("you cannot swipe on yourself")
	}
	if _, err := s.completeViewer(ctx, userID); err != nil {
		return SwipeResponse{}, err
	}
	s.repo.TouchActive(ctx, userID)

	out, err := s.repo.Swipe(ctx, userID, target, req.Action)
	switch {
	case errors.Is(err, ErrTargetUnavailable):
		return SwipeResponse{}, httpx.NotFound("user not found")
	case errors.Is(err, ErrNotFound):
		return SwipeResponse{}, httpx.Conflict("swipe already recorded, retry")
	case err != nil:
		return SwipeResponse{}, err
	}
	if !out.Matched {
		return SwipeResponse{Matched: false}, nil
	}

	now := s.now().UTC()
	mine, err := s.repo.MatchRow(ctx, userID, out.MatchID)
	if err != nil {
		return SwipeResponse{}, err
	}
	summary := matches.Finalize(mine, now)
	resp := SwipeResponse{Matched: true, Match: &summary}

	if out.Created {
		// Events go out only for the call that created the match.
		s.pub.Publish(userID, events.MatchNew, summary)
		if theirs, err := s.repo.MatchRow(ctx, target, out.MatchID); err == nil {
			s.pub.Publish(target, events.MatchNew, matches.Finalize(theirs, now))
		}
		for uid, n := range out.Notifications {
			s.pub.Publish(uid, events.NotificationNew, n)
		}
	}
	return resp, nil
}
