package matching

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo      *Repository
	publisher realtime.Publisher
}

func NewService(repo *Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.Card, error) {
	return s.repo.Discover(ctx, userID, limit)
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResult, error) {
	if req.Action != "like" && req.Action != "pass" {
		return SwipeResult{}, platformerrors.Invalid("action must be like or pass")
	}
	if req.TargetID == userID {
		return SwipeResult{}, platformerrors.Invalid("you cannot swipe yourself")
	}
	res, err := s.repo.Swipe(ctx, userID, req.TargetID, req.Action)
	switch {
	case errors.Is(err, ErrAlreadySwiped):
		return SwipeResult{}, platformerrors.Conflict("already_swiped", "you already decided on this profile")
	case errors.Is(err, ErrTargetUnavailable):
		return SwipeResult{}, platformerrors.ErrNotFound
	case errors.Is(err, ErrIncompleteProfile):
		return SwipeResult{}, platformerrors.Conflict("profile_incomplete", "complete your profile (details, preferences, a photo) first")
	case err != nil:
		return SwipeResult{}, err
	}
	if res.Matched {
		ev := realtime.Event{Type: "match", Data: map[string]string{"matchId": res.MatchID, "conversationId": res.ConversationID}}
		s.publisher.Publish(userID, ev)
		s.publisher.Publish(req.TargetID, ev)
	}
	return res, nil
}

func (s *Service) Matches(ctx context.Context, userID string, before time.Time, limit int) ([]Match, error) {
	return s.repo.ListMatches(ctx, userID, before, limit)
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if errors.Is(err, ErrMatchNotFound) {
		return platformerrors.ErrNotFound
	}
	if err != nil {
		return err
	}
	s.publisher.Publish(other, realtime.Event{Type: "unmatch", Data: map[string]string{"matchId": matchID}})
	return nil
}

func (s *Service) LikesReceived(ctx context.Context, userID string, before time.Time, limit int) ([]LikeReceived, error) {
	return s.repo.LikesReceived(ctx, userID, before, limit)
}
