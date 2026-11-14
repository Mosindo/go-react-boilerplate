package matching

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var ErrInvalidSwipe = errors.New("invalid swipe")

// CardProvider hydrates public profiles (implemented by profiles.Service).
type CardProvider interface {
	Cards(ctx context.Context, viewerID string, userIDs []string) ([]profiles.PublicProfile, error)
}

// MatchNotifier is implemented by notifications.Service.
type MatchNotifier interface {
	NotifyMatch(ctx context.Context, userID, matchID, otherUserID, otherFirstName string)
}

type Service struct {
	repo     Repository
	cards    CardProvider
	notifier MatchNotifier
	pub      realtime.Publisher
}

func NewService(repo Repository, cards CardProvider, notifier MatchNotifier, pub realtime.Publisher) *Service {
	return &Service{repo: repo, cards: cards, notifier: notifier, pub: pub}
}

func (s *Service) requireComplete(ctx context.Context, userID string) error {
	ok, err := s.repo.ProfileComplete(ctx, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrProfileIncomplete
	}
	return nil
}

func (s *Service) Discover(ctx context.Context, viewerID string, limit int) ([]profiles.PublicProfile, error) {
	if err := s.requireComplete(ctx, viewerID); err != nil {
		return nil, err
	}
	ids, err := s.repo.Candidates(ctx, viewerID, limit)
	if err != nil {
		return nil, err
	}
	return s.cards.Cards(ctx, viewerID, ids)
}

func (s *Service) Swipe(ctx context.Context, actorID, targetID, action string) (SwipeResponse, error) {
	if action != ActionLike && action != ActionPass {
		return SwipeResponse{}, ErrInvalidSwipe
	}
	if actorID == targetID {
		return SwipeResponse{}, ErrInvalidSwipe
	}
	if err := s.requireComplete(ctx, actorID); err != nil {
		return SwipeResponse{}, err
	}
	result, err := s.repo.Swipe(ctx, actorID, targetID, action)
	if err != nil {
		return SwipeResponse{}, err
	}
	if result.MatchID == "" {
		return SwipeResponse{Matched: false}, nil
	}

	cards, err := s.cards.Cards(ctx, actorID, []string{targetID, actorID})
	if err != nil {
		return SwipeResponse{}, err
	}
	var targetCard, actorCard profiles.PublicProfile
	for _, c := range cards {
		if c.UserID == targetID {
			targetCard = c
		} else {
			actorCard = c
		}
	}
	match := &Match{ID: result.MatchID, User: targetCard, CreatedAt: result.MatchCreatedAt}

	// The other member learns about the match in real time and through a
	// notification; the actor sees it directly in the response.
	s.notifier.NotifyMatch(ctx, targetID, result.MatchID, actorID, actorCard.FirstName)
	s.notifier.NotifyMatch(ctx, actorID, result.MatchID, targetID, targetCard.FirstName)
	if s.pub != nil {
		s.pub.Publish(targetID, realtime.Event{Type: "match.created", Data: Match{ID: result.MatchID, User: actorCard, CreatedAt: result.MatchCreatedAt}})
	}
	return SwipeResponse{Matched: true, Match: match}, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	if s.pub != nil {
		s.pub.Publish(other, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	}
	return nil
}
