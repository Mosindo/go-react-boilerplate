package matching

import (
	"context"
	"log"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

// CardLoader is satisfied by profiles.Service.
type CardLoader interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Card, error)
}

// MatchNotifier is satisfied by notifications.Service.
type MatchNotifier interface {
	NotifyMatch(ctx context.Context, userID, actorID, conversationID string) error
}

type Service struct {
	repo        Repository
	cards       CardLoader
	notifier    MatchNotifier
	publisher   realtime.Publisher
	ranker      Ranker
	includeDemo bool
	now         func() time.Time
}

type Config struct {
	// IncludeDemo exposes seeded demo accounts in discovery (never in production).
	IncludeDemo bool
}

func NewService(repo Repository, cards CardLoader, notifier MatchNotifier, publisher realtime.Publisher, cfg Config) *Service {
	return &Service{
		repo:        repo,
		cards:       cards,
		notifier:    notifier,
		publisher:   publisher,
		ranker:      DefaultRanker{},
		includeDemo: cfg.IncludeDemo,
		now:         time.Now,
	}
}

// Discover returns the next profiles to review. Already-swiped profiles never come back.
func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.Card, error) {
	viewer, err := s.repo.LoadViewer(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !viewer.HasPhoto {
		return nil, ErrProfileIncomplete
	}
	if err := s.repo.TouchActivity(ctx, userID); err != nil {
		log.Printf(`{"event":"touch_activity_failed","error":%q}`, err.Error())
	}

	now := s.now()
	pool, err := s.repo.Candidates(ctx, CandidateQuery{
		Viewer:      viewer,
		Now:         now,
		Limit:       min(limit*5, 100),
		IncludeDemo: s.includeDemo,
	})
	if err != nil {
		return nil, err
	}
	ranked := s.ranker.Rank(viewer, pool, now)
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	ids := make([]string, len(ranked))
	for i, c := range ranked {
		ids[i] = c.UserID
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

func (s *Service) Swipe(ctx context.Context, userID, targetID, decision string) (SwipeResponse, error) {
	if userID == targetID {
		return SwipeResponse{}, ErrSelfSwipe
	}
	viewer, err := s.repo.LoadViewer(ctx, userID)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !viewer.HasPhoto {
		return SwipeResponse{}, ErrProfileIncomplete
	}

	result, err := s.repo.Swipe(ctx, userID, targetID, decision)
	if err != nil {
		return SwipeResponse{}, err
	}
	if !result.Matched {
		return SwipeResponse{Matched: false}, nil
	}

	cards, err := s.cards.Cards(ctx, userID, []string{userID, targetID})
	if err != nil {
		return SwipeResponse{}, err
	}
	s.announceMatch(ctx, result, userID, targetID, cards)

	return SwipeResponse{
		Matched: true,
		Match:   &MatchResponse{ID: result.MatchID, ConversationID: result.ConversationID, User: cards[targetID]},
	}, nil
}

// announceMatch notifies both people. Failures are logged: the match itself is committed.
func (s *Service) announceMatch(ctx context.Context, result SwipeResult, userID, targetID string, cards map[string]profiles.Card) {
	for _, pair := range [][2]string{{userID, targetID}, {targetID, userID}} {
		recipient, partner := pair[0], pair[1]
		if err := s.notifier.NotifyMatch(ctx, recipient, partner, result.ConversationID); err != nil {
			log.Printf(`{"event":"match_notification_failed","error":%q}`, err.Error())
		}
		s.publisher.Publish(recipient, realtime.Event{
			Type: "match.new",
			Payload: MatchResponse{
				ID:             result.MatchID,
				ConversationID: result.ConversationID,
				User:           cards[partner],
			},
		})
	}
}

func (s *Service) LikesReceived(ctx context.Context, userID string, limit int) ([]profiles.Card, error) {
	ids, err := s.repo.LikesReceived(ctx, userID, limit, s.includeDemo)
	if err != nil {
		return nil, err
	}
	cards, err := s.cards.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	out := make([]profiles.Card, 0, len(ids))
	for _, id := range ids {
		if c, ok := cards[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	partnerID, conversationID, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	event := realtime.Event{Type: "conversation.closed", Payload: map[string]string{"conversationId": conversationID, "matchId": matchID}}
	s.publisher.Publish(partnerID, event)
	s.publisher.Publish(userID, event)
	return nil
}
