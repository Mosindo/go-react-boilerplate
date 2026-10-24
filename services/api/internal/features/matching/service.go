package matching

import (
	"context"
	"log"

	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

// CardLoader is implemented by profiles.Service.
type CardLoader interface {
	Cards(ctx context.Context, viewerID string, ids []string) (map[string]profiles.Card, error)
}

type Service struct {
	repo     Repository
	cards    CardLoader
	notifier notifications.Notifier
	pub      realtime.Publisher
}

func NewService(repo Repository, cards CardLoader, notifier notifications.Notifier, pub realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, notifier: notifier, pub: pub}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.Card, error) {
	if limit < 1 {
		limit = DefaultDiscoverLimit
	}
	if limit > MaxDiscoverLimit {
		limit = MaxDiscoverLimit
	}
	ids, err := s.repo.Discover(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]profiles.Card, 0, len(ids))
	for _, id := range ids { // keep the ranking order
		if c, ok := cards[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *Service) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, error) {
	if userID == targetID {
		return SwipeResult{}, ErrNotEligible
	}
	outcome, err := s.repo.Swipe(ctx, userID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	result := SwipeResult{Action: action, Matched: outcome.Matched}
	if !outcome.Matched {
		return result, nil
	}

	mine, err := s.cards.Cards(ctx, userID, []string{targetID})
	if err != nil {
		return SwipeResult{}, err
	}
	if card, ok := mine[targetID]; ok {
		result.Match = &MatchInfo{ID: outcome.MatchID, ConversationID: outcome.ConversationID, User: card}
	}
	if outcome.NewMatch {
		s.announce(ctx, userID, targetID, outcome)
	}
	return result, nil
}

// announce notifies both people (persisted notification + live event). Failures are logged, never returned.
func (s *Service) announce(ctx context.Context, userID, targetID string, o swipeOutcome) {
	for _, pair := range [][2]string{{userID, targetID}, {targetID, userID}} {
		viewer, other := pair[0], pair[1]
		cards, err := s.cards.Cards(ctx, viewer, []string{other})
		if err != nil || cards[other].UserID == "" {
			log.Printf(`{"event":"match_announce_failed","viewer":%q}`, viewer)
			continue
		}
		card := cards[other]
		s.pub.Publish(viewer, realtime.Event{Type: "match.new", Data: MatchInfo{ID: o.MatchID, ConversationID: o.ConversationID, User: card}})
		s.notifier.Notify(viewer, notifications.New{
			Type:  notifications.TypeMatch,
			Title: "C'est un match !",
			Body:  "Vous et " + card.FirstName + " vous plaisez mutuellement. Dites bonjour !",
			Data:  map[string]any{"matchId": o.MatchID, "conversationId": o.ConversationID, "userId": other},
		})
	}
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	u, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.pub.Publish(u.OtherUserID, realtime.Event{Type: "match.removed", Data: map[string]string{"conversationId": u.ConversationID}})
	return nil
}
