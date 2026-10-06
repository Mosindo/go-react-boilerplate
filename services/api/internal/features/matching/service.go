package matching

import (
	"context"
	"sort"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

// Notifier creates user notifications (implemented by notifications.Service).
type Notifier interface {
	Notify(ctx context.Context, n notifications.New)
}

// Cards are the discovery view of a user: public profile plus the interests
// shared with the viewer.
type Card struct {
	Profile profiles.PublicProfile
	Shared  map[int16]struct{}
}

type Service struct {
	repo     Repository
	profiles profiles.Repository
	ranker   Ranker
	notifier Notifier
	hub      *realtime.Hub
	now      func() time.Time
}

func NewService(repo Repository, profileRepo profiles.Repository, ranker Ranker, notifier Notifier, hub *realtime.Hub) *Service {
	return &Service{repo: repo, profiles: profileRepo, ranker: ranker, notifier: notifier, hub: hub, now: time.Now}
}

// Discover returns the next profiles to show: filtered in SQL, ranked here.
// Profiles already swiped never come back, so the client simply asks again.
func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]Card, error) {
	if limit <= 0 {
		limit = defaultDiscoverLimit
	}
	if limit > maxDiscoverLimit {
		limit = maxDiscoverLimit
	}
	viewer, err := s.repo.Viewer(ctx, userID)
	if err != nil {
		return nil, err
	}
	candidates, err := s.repo.Candidates(ctx, userID, candidatePoolSize)
	if err != nil || len(candidates) == 0 {
		return []Card{}, err
	}
	return s.rankAndHydrate(ctx, viewer, candidates, limit)
}

func (s *Service) LikesReceived(ctx context.Context, userID string, limit, offset int) ([]Card, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	viewer, err := s.repo.Viewer(ctx, userID)
	if err != nil {
		return nil, err
	}
	candidates, err := s.repo.LikesReceived(ctx, userID, limit, offset)
	if err != nil || len(candidates) == 0 {
		return []Card{}, err
	}
	return s.hydrate(ctx, viewer, candidates)
}

func (s *Service) rankAndHydrate(ctx context.Context, viewer Viewer, candidates []Candidate, limit int) ([]Card, error) {
	ids := make([]string, len(candidates))
	for i, c := range candidates {
		ids[i] = c.UserID
	}
	interests, err := s.profiles.GetInterests(ctx, ids)
	if err != nil {
		return nil, err
	}

	now := s.now()
	type scored struct {
		c     Candidate
		score float64
	}
	ranked := make([]scored, len(candidates))
	for i, c := range candidates {
		shared := 0
		for _, in := range interests[c.UserID] {
			if _, ok := viewer.Interests[in.ID]; ok {
				shared++
			}
		}
		ranked[i] = scored{c: c, score: s.ranker.Score(viewer, c, shared, now)}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].c.UserID < ranked[j].c.UserID
	})
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	top := make([]Candidate, len(ranked))
	for i, r := range ranked {
		top[i] = r.c
	}
	return s.hydrateWithInterests(ctx, viewer, top, interests)
}

func (s *Service) hydrate(ctx context.Context, viewer Viewer, candidates []Candidate) ([]Card, error) {
	ids := make([]string, len(candidates))
	for i, c := range candidates {
		ids[i] = c.UserID
	}
	interests, err := s.profiles.GetInterests(ctx, ids)
	if err != nil {
		return nil, err
	}
	return s.hydrateWithInterests(ctx, viewer, candidates, interests)
}

func (s *Service) hydrateWithInterests(ctx context.Context, viewer Viewer, candidates []Candidate, interests map[string][]profiles.Interest) ([]Card, error) {
	ids := make([]string, len(candidates))
	for i, c := range candidates {
		ids[i] = c.UserID
	}
	photos, err := s.profiles.GetPhotos(ctx, ids)
	if err != nil {
		return nil, err
	}
	now := s.now()
	cards := make([]Card, 0, len(candidates))
	for _, c := range candidates {
		pp := profiles.PublicProfile{
			UserID:    c.UserID,
			FirstName: c.FirstName,
			Gender:    c.Gender,
			Bio:       c.Bio,
			Interests: interests[c.UserID],
			Photos:    photos[c.UserID],
		}
		if c.ShowAge {
			age := profiles.AgeOn(c.Birthdate, now)
			pp.Age = &age
		}
		// City and distance are one privacy setting: both are location hints.
		if c.ShowDistance {
			pp.City = c.City
			if c.DistanceKm != nil {
				km := profiles.ApproxDistanceKm(*c.DistanceKm)
				pp.DistanceKm = &km
			}
		}
		shared := map[int16]struct{}{}
		for _, in := range pp.Interests {
			if _, ok := viewer.Interests[in.ID]; ok {
				shared[in.ID] = struct{}{}
			}
		}
		cards = append(cards, Card{Profile: pp, Shared: shared})
	}
	return cards, nil
}

// Swipe records a like or pass. A reciprocal like creates the match, notifies
// both users and pushes a realtime event to each.
func (s *Service) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, error) {
	if action != ActionLike && action != ActionPass {
		return SwipeResult{}, ErrInvalidAction
	}
	if userID == targetID {
		return SwipeResult{}, ErrSelfSwipe
	}
	if _, err := s.repo.Viewer(ctx, userID); err != nil {
		return SwipeResult{}, err
	}
	res, err := s.repo.Swipe(ctx, userID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	if res.NewMatch {
		s.announceMatch(ctx, userID, targetID, res)
	}
	return res, nil
}

func (s *Service) announceMatch(ctx context.Context, userID, targetID string, res SwipeResult) {
	for _, side := range []struct{ to, other, otherName string }{
		{to: userID, other: targetID, otherName: res.OtherName},
		{to: targetID, other: userID, otherName: res.SelfName},
	} {
		data := map[string]string{"matchId": res.Match.ID, "userId": side.other}
		s.notifier.Notify(ctx, notifications.New{
			UserID: side.to,
			Type:   notifications.TypeNewMatch,
			Title:  "C'est un match !",
			Body:   "Vous et " + side.otherName + " vous plaisez mutuellement. Dites bonjour !",
			Data:   data,
		})
		s.hub.Publish(side.to, realtime.Event{Type: "match.new", Data: data})
	}
}
