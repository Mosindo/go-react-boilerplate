package discovery

import (
	"context"
	"errors"

	perr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

// Notifier creates in-app notifications (implemented by notifications.Service).
type Notifier interface {
	Create(ctx context.Context, userID, typ string, data map[string]any) error
}

type Service struct {
	repo     *Repository
	notifier Notifier
	hub      *realtime.Hub
}

func NewService(repo *Repository, notifier Notifier, hub *realtime.Hub) *Service {
	return &Service{repo: repo, notifier: notifier, hub: hub}
}

const (
	DefaultDeck = 10
	MaxDeck     = 20
)

func (s *Service) Deck(ctx context.Context, userID string, limit int) ([]Card, error) {
	if limit <= 0 {
		limit = DefaultDeck
	}
	if limit > MaxDeck {
		limit = MaxDeck
	}
	return s.repo.Deck(ctx, userID, limit)
}

func (s *Service) Profile(ctx context.Context, viewerID, targetID string) (Card, error) {
	if viewerID == targetID {
		return Card{}, perr.BadRequest("invalid_target", "C'est votre propre profil.")
	}
	card, err := s.repo.Profile(ctx, viewerID, targetID)
	if errors.Is(err, ErrNotEligible) {
		return Card{}, perr.NotFound("Profil introuvable.")
	}
	return card, err
}

func (s *Service) Swipe(ctx context.Context, viewerID, targetID, action string) (SwipeResult, error) {
	if viewerID == targetID {
		return SwipeResult{}, perr.BadRequest("invalid_target", "Vous ne pouvez pas vous liker vous-même.")
	}
	res, err := s.repo.Swipe(ctx, viewerID, targetID, action)
	switch {
	case errors.Is(err, ErrNotEligible):
		return SwipeResult{}, perr.NotFound("Profil introuvable.")
	case errors.Is(err, ErrIncomplete):
		return SwipeResult{}, perr.Conflict("incomplete_profile", "Complétez votre profil (photo et position) pour continuer.")
	case errors.Is(err, ErrAlreadyDecided):
		return SwipeResult{}, perr.Conflict("already_decided", "Vous avez déjà pris une décision pour ce profil.")
	case err != nil:
		return SwipeResult{}, err
	}
	if res.Matched {
		s.announceMatch(ctx, res.MatchID, viewerID, targetID)
	}
	return res, nil
}

// announceMatch notifies both users. It runs after the transaction committed;
// a failure here must not undo the match, so errors are only recorded.
func (s *Service) announceMatch(ctx context.Context, matchID, a, b string) {
	for _, pair := range [][2]string{{a, b}, {b, a}} {
		to, other := pair[0], pair[1]
		card, err := s.repo.Profile(ctx, to, other)
		if err != nil {
			continue
		}
		data := map[string]any{"matchId": matchID, "userId": other, "firstName": card.FirstName}
		if len(card.Photos) > 0 {
			data["photoUrl"] = card.Photos[0].URL
		}
		_ = s.notifier.Create(ctx, to, "match", data)
		s.hub.Publish(to, realtime.Event{Type: "match.new", Data: data})
	}
}
