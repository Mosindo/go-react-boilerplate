package matching

import (
	"context"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

// Hydrator attaches interests and photos to raw profile cards.
type Hydrator interface {
	Hydrate(ctx context.Context, cards []profiles.Card) ([]profiles.PublicProfile, error)
}

type Service struct {
	repo      Repository
	hydrator  Hydrator
	ranker    Ranker
	publisher realtime.Publisher
}

func NewService(repo Repository, hydrator Hydrator, ranker Ranker, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, hydrator: hydrator, ranker: ranker, publisher: publisher}
}

func cardOf(c candidate) profiles.Card {
	card := c.Card
	if c.ShowDistance {
		km := c.DistanceKm
		card.DistanceKm = &km
	}
	return card
}

// Discover returns the next batch of profiles. Candidates come from a bounded
// pool (nearest first, already filtered and de-duplicated by the repository)
// that the Ranker reorders; only the requested page is hydrated.
func (s *Service) Discover(ctx context.Context, viewerID string, limit int) ([]profiles.PublicProfile, error) {
	ready, err := s.repo.ViewerReady(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if !ready {
		return nil, ErrProfileIncomplete
	}
	pool, err := s.repo.Candidates(ctx, viewerID, limit*4)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(pool))
	for i, c := range pool {
		ids[i] = c.Card.UserID
	}
	shared, err := s.repo.SharedInterests(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	for i := range pool {
		pool[i].SharedInterests = shared[pool[i].Card.UserID]
	}
	ranked := s.ranker.Rank(pool)
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	cards := make([]profiles.Card, len(ranked))
	for i, c := range ranked {
		cards[i] = cardOf(c)
	}
	return s.hydrator.Hydrate(ctx, cards)
}

func (s *Service) Swipe(ctx context.Context, viewerID, targetID, action string) (SwipeResponse, error) {
	if viewerID == targetID {
		return SwipeResponse{}, ErrNotFound
	}
	ready, err := s.repo.ViewerReady(ctx, viewerID)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !ready {
		return SwipeResponse{}, ErrProfileIncomplete
	}
	out, err := s.repo.Swipe(ctx, viewerID, targetID, action)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !out.Matched {
		return SwipeResponse{Matched: false}, nil
	}

	profile, err := s.Profile(ctx, viewerID, targetID)
	if err != nil {
		return SwipeResponse{}, err
	}
	// The person who just swiped learns about the match from this response; only
	// the other side needs a live event.
	s.publisher.Publish(targetID, realtime.Event{Type: "match.new", Data: map[string]string{
		"matchId": out.MatchID, "conversationId": out.ConversationID, "userId": viewerID, "firstName": out.ViewerName}})
	return SwipeResponse{Matched: true, Match: &MatchInfo{
		ID: out.MatchID, ConversationID: out.ConversationID, CreatedAt: out.CreatedAt, User: profile,
	}}, nil
}

// Profile returns the public view of a target the viewer is allowed to see.
func (s *Service) Profile(ctx context.Context, viewerID, targetID string) (profiles.PublicProfile, error) {
	card, km, err := s.repo.GetCard(ctx, viewerID, targetID)
	if err != nil {
		return profiles.PublicProfile{}, err
	}
	card.DistanceKm = km
	list, err := s.hydrator.Hydrate(ctx, []profiles.Card{card})
	if err != nil {
		return profiles.PublicProfile{}, err
	}
	return list[0], nil
}

func (s *Service) ResetPasses(ctx context.Context, viewerID string) (int64, error) {
	return s.repo.ResetPasses(ctx, viewerID)
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.publisher.Publish(other, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	return nil
}
