package matching

import (
	"context"
	"log"

	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo Repository
	rt   realtime.Publisher
}

func NewService(repo Repository, rt realtime.Publisher) *Service { return &Service{repo: repo, rt: rt} }

// Feed renvoie un lot borne de profils non encore traites. Les distances sont
// arrondies (jamais de coordonnees) et masquees si l'utilisateur l'a demande.
func (s *Service) Feed(ctx context.Context, userID string, limit int) ([]Card, error) {
	cards, err := s.repo.Feed(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	for i := range cards {
		if cards[i].showDistance {
			d := geo.BucketKm(cards[i].distanceExactKm)
			cards[i].DistanceKm = &d
		}
	}
	return cards, nil
}

func (s *Service) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, error) {
	if action != ActionLike && action != ActionPass {
		return SwipeResult{}, ErrInvalidAction
	}
	if userID == targetID {
		return SwipeResult{}, ErrNotFound
	}
	res, pair, err := s.repo.Swipe(ctx, userID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	if pair != nil {
		// Temps reel apres commit: chaque participant recoit les infos de l'autre.
		s.rt.Publish(userID, realtime.Event{Type: "match", Payload: pair.Self})
		s.rt.Publish(targetID, realtime.Event{Type: "match", Payload: pair.Other})
	}
	return res, nil
}

func (s *Service) ListMatches(ctx context.Context, userID string, limit int, cur *httpx.Cursor) ([]MatchSummary, error) {
	return s.repo.ListMatches(ctx, userID, limit, cur)
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.rt.Publish(other, realtime.Event{Type: "unmatch", Payload: map[string]string{"matchId": matchID}})
	log.Printf(`{"event":"unmatch","match_id":%q}`, matchID)
	return nil
}
