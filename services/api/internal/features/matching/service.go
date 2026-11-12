package matching

import (
	"context"
	"math"
	"slices"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

const (
	defaultDiscoverLimit = 10
	maxDiscoverLimit     = 30
	maxExcluded          = 50
)

type ProfileProvider interface {
	IsComplete(ctx context.Context, userID string) (bool, error)
	PublicProfiles(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
}

type Notifier interface {
	Notify(ctx context.Context, userID, kind, title, body string, data map[string]string) error
}

type Service struct {
	repo      Repository
	profiles  ProfileProvider
	notifier  Notifier
	publisher realtime.Publisher
	now       func() time.Time
}

func NewService(repo Repository, profiles ProfileProvider, notifier Notifier, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profiles, notifier: notifier, publisher: publisher, now: time.Now}
}

// buildDiscoverQuery turns the viewer's own profile and preferences into the
// candidate filter. Exposed for unit tests.
func buildDiscoverQuery(viewerID string, v viewerContext, exclude []string, limit int, now time.Time) discoverQuery {
	earliest, latest := profiles.BirthdateBounds(v.MinAge, v.MaxAge, now)
	q := discoverQuery{
		ViewerID:       viewerID,
		OrganizationID: v.OrganizationID,
		Gender:         v.Gender,
		Age:            profiles.AgeOn(v.Birthdate, now),
		InterestedIn:   v.InterestedIn,
		Earliest:       earliest,
		Latest:         latest,
		MaxDistanceKm:  v.MaxDistanceKm,
		Exclude:        exclude,
		Limit:          limit,
	}
	if v.Latitude != nil && v.Longitude != nil {
		dist := float64(v.MaxDistanceKm)
		dLat := dist/111.0 + 0.02
		q.Latitude, q.Longitude, q.DeltaLat = v.Latitude, v.Longitude, &dLat
		if c := math.Cos(*v.Latitude * math.Pi / 180); c > 0.05 {
			dLng := dist/(111.32*c) + 0.02
			q.DeltaLng = &dLng
		}
	}
	return q
}

func (s *Service) Discover(ctx context.Context, viewerID string, exclude []string, limit int) ([]profiles.PublicProfile, error) {
	complete, err := s.profiles.IsComplete(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if !complete {
		return nil, ErrProfileIncomplete
	}
	v, err := s.repo.ViewerContext(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = defaultDiscoverLimit
	}
	limit = min(limit, maxDiscoverLimit)
	if len(exclude) > maxExcluded {
		exclude = exclude[:maxExcluded]
	}
	if exclude == nil {
		exclude = []string{}
	}
	ids, err := s.repo.Discover(ctx, buildDiscoverQuery(viewerID, v, exclude, limit, s.now()))
	if err != nil {
		return nil, err
	}
	return s.profiles.PublicProfiles(ctx, viewerID, ids)
}

func (s *Service) Swipe(ctx context.Context, actorID, organizationID, targetID, decision string) (SwipeResponse, error) {
	if !slices.Contains([]string{DecisionLike, DecisionPass}, decision) {
		return SwipeResponse{}, ErrInvalidDecision
	}
	if actorID == targetID {
		return SwipeResponse{}, ErrSelfSwipe
	}
	complete, err := s.profiles.IsComplete(ctx, actorID)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !complete {
		return SwipeResponse{}, ErrProfileIncomplete
	}

	res, err := s.repo.Swipe(ctx, actorID, organizationID, targetID, decision)
	if err != nil {
		return SwipeResponse{}, err
	}
	out := SwipeResponse{Decision: decision, Matched: res.Matched}
	if !res.Matched {
		return out, nil
	}

	cards, err := s.profiles.PublicProfiles(ctx, actorID, []string{actorID, targetID})
	if err != nil {
		return SwipeResponse{}, err
	}
	var actorCard, targetCard profiles.PublicProfile
	for _, c := range cards {
		if c.UserID == actorID {
			actorCard = c
		} else {
			targetCard = c
		}
	}
	out.Match = &Match{ID: res.MatchID, User: targetCard, CreatedAt: res.CreatedAt}

	s.announceMatch(ctx, actorID, targetCard, res)
	s.announceMatch(ctx, targetID, actorCard, res)
	return out, nil
}

func (s *Service) announceMatch(ctx context.Context, recipientID string, other profiles.PublicProfile, res swipeResult) {
	data := map[string]string{"matchId": res.MatchID, "userId": other.UserID}
	_ = s.notifier.Notify(ctx, recipientID, "match", "C'est un match !", other.FirstName+" et vous vous plaisez mutuellement. Dites bonjour !", data)
	s.publisher.Publish(recipientID, realtime.Event{Type: "match.new", Data: data})
}

func (s *Service) ListMatches(ctx context.Context, userID string, limit, offset int) ([]Match, error) {
	stored, err := s.repo.ListMatches(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(stored))
	for _, m := range stored {
		ids = append(ids, m.OtherUserID)
	}
	cards, err := s.profiles.PublicProfiles(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.PublicProfile, len(cards))
	for _, c := range cards {
		byID[c.UserID] = c
	}
	out := make([]Match, 0, len(stored))
	for _, m := range stored {
		card, ok := byID[m.OtherUserID]
		if !ok {
			continue
		}
		out = append(out, Match{ID: m.ID, User: card, CreatedAt: m.CreatedAt, LastMessageAt: m.LastMessageAt})
	}
	return out, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	otherID, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.publisher.Publish(otherID, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	s.publisher.Publish(userID, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	return nil
}
