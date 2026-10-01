package discovery

import (
	"context"
	"log"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/realtime"
)

const (
	DefaultBatch = 10
	MaxBatch     = 20
)

// Cards is the slice of the profiles service discovery needs.
type Cards interface {
	Cards(ctx context.Context, viewerID string, ids []string) (map[string]profiles.Card, error)
}

// Notifier records an in-app notification (and pushes it in real time).
type Notifier interface {
	Notify(ctx context.Context, userID, kind, actorID, refID string) error
}

// Publisher pushes realtime events to connected clients.
type Publisher interface {
	Publish(userID string, ev realtime.Event)
}

type Service struct {
	repo     *Repository
	cards    Cards
	notifier Notifier
	pub      Publisher
}

func NewService(repo *Repository, cards Cards, notifier Notifier, pub Publisher) *Service {
	return &Service{repo: repo, cards: cards, notifier: notifier, pub: pub}
}

func (s *Service) requireComplete(ctx context.Context, userID string) error {
	ok, err := s.repo.HasCompleteProfile(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrProfileIncomplete
	}
	return nil
}

// Feed returns the next batch of profiles. Already-swiped users are excluded server-side, so
// calling it again after acting on the batch yields fresh profiles without any cursor.
func (s *Service) Feed(ctx context.Context, userID string, limit int) ([]profiles.Card, error) {
	if limit <= 0 {
		limit = DefaultBatch
	}
	if limit > MaxBatch {
		limit = MaxBatch
	}
	if err := s.requireComplete(ctx, userID); err != nil {
		return nil, err
	}
	ids, err := s.repo.Candidates(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	return s.orderedCards(ctx, userID, ids)
}

func (s *Service) orderedCards(ctx context.Context, viewerID string, ids []string) ([]profiles.Card, error) {
	byID, err := s.cards.Cards(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]profiles.Card, 0, len(ids))
	for _, id := range ids {
		if c, ok := byID[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResult, error) {
	if req.Action != ActionLike && req.Action != ActionPass {
		return SwipeResult{}, ErrInvalidAction
	}
	if !httpx.IsUUID(req.UserID) {
		return SwipeResult{}, ErrNotFound
	}
	if req.UserID == userID {
		return SwipeResult{}, ErrSelf
	}
	if err := s.requireComplete(ctx, userID); err != nil {
		return SwipeResult{}, err
	}
	res, err := s.repo.Swipe(ctx, userID, req.UserID, req.Action)
	if err != nil {
		return SwipeResult{}, err
	}
	if res.Matched {
		s.announceMatch(ctx, userID, req.UserID, &res)
	}
	return res, nil
}

// announceMatch notifies both users and attaches the other user's card to the swiper's result.
// Notification failures are logged, never surfaced: the match itself is already committed.
func (s *Service) announceMatch(ctx context.Context, userID, otherID string, res *SwipeResult) {
	for _, pair := range [][2]string{{userID, otherID}, {otherID, userID}} {
		recipient, other := pair[0], pair[1]
		if err := s.notifier.Notify(ctx, recipient, "match", other, res.MatchID); err != nil {
			log.Printf("discovery: notify match: %v", err)
		}
		cards, err := s.cards.Cards(ctx, recipient, []string{other})
		if err != nil {
			log.Printf("discovery: match card: %v", err)
			continue
		}
		card, ok := cards[other]
		if !ok {
			continue
		}
		if recipient == userID {
			// The swiper learns about the match from the HTTP response itself.
			c := card
			res.Match = &c
			continue
		}
		s.pub.Publish(recipient, realtime.Event{Type: "match.new", Data: Match{
			ID: res.MatchID, ConversationID: res.ConversationID, User: card,
		}})
	}
}

func (s *Service) Unswipe(ctx context.Context, userID, targetID string) error {
	if !httpx.IsUUID(targetID) {
		return ErrNotFound
	}
	return s.repo.Unswipe(ctx, userID, targetID)
}

// Profile returns another user's full public profile when the viewer is allowed to see it.
func (s *Service) Profile(ctx context.Context, viewerID, targetID string) (profiles.Card, error) {
	if !httpx.IsUUID(targetID) {
		return profiles.Card{}, ErrNotFound
	}
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return profiles.Card{}, err
	}
	if !ok {
		return profiles.Card{}, ErrNotFound
	}
	cards, err := s.cards.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return profiles.Card{}, err
	}
	c, found := cards[targetID]
	if !found {
		return profiles.Card{}, ErrNotFound
	}
	return c, nil
}

func (s *Service) Matches(ctx context.Context, userID string, limit, offset int) ([]Match, error) {
	rows, err := s.repo.ListMatches(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(rows))
	for _, m := range rows {
		ids = append(ids, m.OtherID)
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]Match, 0, len(rows))
	for _, m := range rows {
		if c, ok := cards[m.OtherID]; ok {
			out = append(out, Match{ID: m.ID, ConversationID: m.ConversationID, CreatedAt: m.CreatedAt, User: c})
		}
	}
	return out, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.pub.Publish(other, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	return nil
}
