package matching

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrProfileIncomplete = errors.New("complete your profile (details, preferences and at least one photo) first")
	ErrSelfSwipe         = errors.New("you cannot swipe yourself")
	ErrAlreadySwiped     = errors.New("you already made a decision about this profile")
	ErrTargetUnavailable = errors.New("this profile is no longer available")
	ErrMatchNotFound     = errors.New("match not found")
)

const maxCandidatePool = 100

// CardLoader hydrates public profile cards (implemented by profiles.Service).
type CardLoader interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]profiles.Card, error)
}

// MatchNotifier records in-app notifications (implemented by notifications.Service).
type MatchNotifier interface {
	NotifyMatch(ctx context.Context, userID, otherFirstName, matchID, conversationID string)
}

type Service struct {
	repo      Repository
	cards     CardLoader
	notifier  MatchNotifier
	publisher realtime.Publisher
	ranker    Ranker
	now       func() time.Time
}

func NewService(repo Repository, cards CardLoader, notifier MatchNotifier, publisher realtime.Publisher, ranker Ranker) *Service {
	return &Service{repo: repo, cards: cards, notifier: notifier, publisher: publisher, ranker: ranker, now: time.Now}
}

func (s *Service) viewer(ctx context.Context, userID string) (ViewerContext, error) {
	viewer, err := s.repo.ViewerContext(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ViewerContext{}, ErrProfileIncomplete
		}
		return ViewerContext{}, err
	}
	if !viewer.HasPhoto {
		return ViewerContext{}, ErrProfileIncomplete
	}
	return viewer, nil
}

// Discover returns the next profiles to review, best first.
func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.Card, error) {
	viewer, err := s.viewer(ctx, userID)
	if err != nil {
		return nil, err
	}
	pool := min(limit*4, maxCandidatePool)
	candidates, err := s.repo.Candidates(ctx, viewer, s.now().UTC(), pool)
	if err != nil {
		return nil, err
	}
	ranked := s.ranker.Rank(candidates, viewer)
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	ids := make([]string, 0, len(ranked))
	for _, c := range ranked {
		ids = append(ids, c.UserID)
	}
	return s.orderedCards(ctx, userID, ids)
}

func (s *Service) orderedCards(ctx context.Context, viewerID string, ids []string) ([]profiles.Card, error) {
	cards, err := s.cards.Cards(ctx, viewerID, ids)
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

// Swipe records a like or a pass and returns the match when it is mutual.
func (s *Service) Swipe(ctx context.Context, userID, targetID, action string) (SwipeResult, *profiles.Card, error) {
	if userID == targetID {
		return SwipeResult{}, nil, ErrSelfSwipe
	}
	if action != ActionLike && action != ActionPass {
		return SwipeResult{}, nil, ErrTargetUnavailable
	}
	if _, err := s.viewer(ctx, userID); err != nil {
		return SwipeResult{}, nil, err
	}
	result, err := s.repo.Swipe(ctx, userID, targetID, action)
	switch {
	case errors.Is(err, ErrRepositoryAlreadySwipe):
		return SwipeResult{}, nil, ErrAlreadySwiped
	case errors.Is(err, ErrRepositoryUnavailable):
		return SwipeResult{}, nil, ErrTargetUnavailable
	case err != nil:
		return SwipeResult{}, nil, err
	}
	if !result.Matched {
		return result, nil, nil
	}

	cards, err := s.cards.Cards(ctx, userID, []string{userID, targetID})
	if err != nil {
		return result, nil, err
	}
	me, other := cards[userID], cards[targetID]
	s.announceMatch(ctx, userID, other, result)
	s.announceMatch(ctx, targetID, me, result)
	return result, &other, nil
}

func (s *Service) announceMatch(ctx context.Context, recipientID string, other profiles.Card, result SwipeResult) {
	s.notifier.NotifyMatch(ctx, recipientID, other.FirstName, result.MatchID, result.ConversationID)
	s.publisher.Publish(recipientID, realtime.Event{Type: "match.created", Data: MatchEvent{
		MatchID:        result.MatchID,
		ConversationID: result.ConversationID,
		User:           profiles.ToCardSummary(other),
	}})
}

func (s *Service) LikesReceived(ctx context.Context, userID string, limit, offset int) ([]profiles.Card, error) {
	ids, err := s.repo.LikesReceived(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	return s.orderedCards(ctx, userID, ids)
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, conversationID, err := s.repo.DeleteMatch(ctx, userID, matchID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrMatchNotFound
		}
		return err
	}
	event := realtime.Event{Type: "match.removed", Data: MatchRemovedEvent{MatchID: matchID, ConversationID: conversationID}}
	s.publisher.Publish(userID, event)
	s.publisher.Publish(other, event)
	return nil
}
