package matching

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrProfileIncomplete = errors.New("profile incomplete")
	ErrSelfSwipe         = errors.New("cannot swipe yourself")
)

type CardProvider interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Card, error)
	TouchActive(ctx context.Context, userID string) error
}

type Notifier interface {
	Notify(ctx context.Context, in notifications.Input)
}

type Service struct {
	repo   Repository
	cards  CardProvider
	notify Notifier
	pub    realtime.Publisher
	ranker Ranker
	now    func() time.Time
}

func NewService(repo Repository, cards CardProvider, notify Notifier, pub realtime.Publisher, ranker Ranker) *Service {
	return &Service{repo: repo, cards: cards, notify: notify, pub: pub, ranker: ranker, now: time.Now}
}

func (s *Service) ensureReady(ctx context.Context, userID string) error {
	r, err := s.repo.Readiness(ctx, userID)
	if err != nil {
		return err
	}
	if !r.HasProfile || !r.HasPreferences || !r.HasPhoto {
		return ErrProfileIncomplete
	}
	return nil
}

// Discover returns up to limit ranked profiles. The candidate pool is larger
// than the page so the ranker has room to reorder.
func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.Card, error) {
	if err := s.ensureReady(ctx, userID); err != nil {
		return nil, err
	}
	_ = s.cards.TouchActive(ctx, userID)

	pool, err := s.repo.Candidates(ctx, userID, min(limit*4, 100))
	if err != nil {
		return nil, err
	}
	ranked := s.ranker.Rank(pool, s.now())
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	ids := make([]string, 0, len(ranked))
	for _, c := range ranked {
		ids = append(ids, c.UserID)
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]profiles.Card, 0, len(ids))
	for _, id := range ids {
		if card, ok := cards[id]; ok {
			out = append(out, card)
		}
	}
	return out, nil
}

func (s *Service) Swipe(ctx context.Context, userID, targetID, direction string) (SwipeResponse, error) {
	if userID == targetID {
		return SwipeResponse{}, ErrSelfSwipe
	}
	if err := s.ensureReady(ctx, userID); err != nil {
		return SwipeResponse{}, err
	}
	match, err := s.repo.RecordSwipe(ctx, userID, targetID, direction)
	if err != nil || match == nil {
		return SwipeResponse{Matched: false}, err
	}

	cardsForMe, err := s.cards.Cards(ctx, userID, []string{targetID})
	if err != nil {
		return SwipeResponse{}, err
	}
	cardsForThem, err := s.cards.Cards(ctx, targetID, []string{userID})
	if err != nil {
		return SwipeResponse{}, err
	}
	mine := MatchPayload{MatchID: match.ID, CreatedAt: match.CreatedAt, User: cardsForMe[targetID]}
	theirs := MatchPayload{MatchID: match.ID, CreatedAt: match.CreatedAt, User: cardsForThem[userID]}

	s.pub.Publish(userID, realtime.Event{Type: realtime.EventMatchCreated, Data: mine})
	s.pub.Publish(targetID, realtime.Event{Type: realtime.EventMatchCreated, Data: theirs})
	s.notifyMatch(ctx, targetID, match.ID, theirs.User.FirstName)
	s.notifyMatch(ctx, userID, match.ID, mine.User.FirstName)

	return SwipeResponse{Matched: true, Match: &mine}, nil
}

func (s *Service) notifyMatch(ctx context.Context, userID, matchID, otherName string) {
	s.notify.Notify(ctx, notifications.Input{
		UserID: userID,
		Type:   notifications.TypeMatch,
		Title:  "Nouveau match !",
		Body:   "Toi et " + otherName + " vous plaisez mutuellement. Dis bonjour !",
		Data:   map[string]string{"matchId": matchID},
	})
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.DeleteMatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	payload := map[string]string{"matchId": matchID}
	s.pub.Publish(other, realtime.Event{Type: realtime.EventMatchRemoved, Data: payload})
	s.pub.Publish(userID, realtime.Event{Type: realtime.EventMatchRemoved, Data: payload})
	return nil
}
