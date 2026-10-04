package discovery

import (
	"context"
	"errors"

	"example.com/api/internal/features/chat"
	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
	"github.com/google/uuid"
)

// MatchReader loads the viewer-relative summary of a match (implemented by the chat feature).
type MatchReader interface {
	MatchSummary(ctx context.Context, userID, matchID string) (*chat.MatchSummary, error)
}

type Service struct {
	repo    Repository
	matches MatchReader
	rt      realtime.Publisher
}

func NewService(repo Repository, matches MatchReader, rt realtime.Publisher) *Service {
	return &Service{repo: repo, matches: matches, rt: rt}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	list, err := s.repo.Discover(ctx, userID, limit)
	if errors.Is(err, ErrNotCompleted) {
		return nil, platformerrors.ProfileIncomplete()
	}
	return list, err
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResponse, error) {
	if req.Action != ActionLike && req.Action != ActionPass {
		return SwipeResponse{}, platformerrors.FieldInvalid("action", "Must be like or pass")
	}
	if _, err := uuid.Parse(req.TargetID); err != nil {
		return SwipeResponse{}, platformerrors.FieldInvalid("targetId", "Invalid user id")
	}
	if req.TargetID == userID {
		return SwipeResponse{}, platformerrors.FieldInvalid("targetId", "You cannot swipe on yourself")
	}

	res, err := s.repo.Swipe(ctx, userID, req.TargetID, req.Action)
	if err != nil {
		switch {
		case errors.Is(err, ErrNotCompleted):
			return SwipeResponse{}, platformerrors.ProfileIncomplete()
		case errors.Is(err, ErrTargetUnavailable):
			return SwipeResponse{}, platformerrors.NotFound("This profile is no longer available")
		}
		return SwipeResponse{}, err
	}

	out := SwipeResponse{Action: res.Action}
	if res.MatchID == "" {
		return out, nil
	}
	mine, err := s.matches.MatchSummary(ctx, userID, res.MatchID)
	if err != nil {
		return SwipeResponse{}, err
	}
	out.Matched = true
	out.Match = mine

	if res.NewMatch {
		if theirs, err := s.matches.MatchSummary(ctx, req.TargetID, res.MatchID); err == nil {
			s.rt.Publish(req.TargetID, "match.new", theirs)
		}
		// The swiper sees the match in the response; other devices of theirs get the event.
		s.rt.Publish(userID, "match.new", mine)
		for _, n := range res.Notifications {
			s.rt.Publish(n.UserID, "notification.new", n.Value)
		}
	}
	return out, nil
}
