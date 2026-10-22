package matching

import (
	"context"
	"time"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo     Repository
	realtime realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, realtime: publisher}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	if limit <= 0 {
		limit = DefaultDiscoverLimit
	}
	if limit > MaxDiscoverLimit {
		limit = MaxDiscoverLimit
	}
	return s.repo.Discover(ctx, userID, limit)
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResult, error) {
	if !profiles.IsUUID(req.UserID) {
		return SwipeResult{}, ErrNotEligible
	}
	if req.Action != ActionLike && req.Action != ActionPass {
		return SwipeResult{}, platformerrors.Invalid("action", "action invalide (like ou pass)")
	}
	if req.UserID == userID {
		return SwipeResult{}, platformerrors.Invalid("userId", "vous ne pouvez pas vous évaluer vous-même")
	}

	outcome, err := s.repo.Swipe(ctx, userID, req.UserID, req.Action)
	if err != nil {
		return SwipeResult{}, err
	}
	result := SwipeResult{Action: outcome.Action, AlreadySwiped: outcome.AlreadySwiped}
	if outcome.MatchID == "" {
		return result, nil
	}

	match, err := s.repo.GetMatch(ctx, userID, outcome.MatchID)
	if err != nil {
		return SwipeResult{}, err
	}
	result.Matched = true
	result.Match = &match

	if outcome.NewMatch {
		// Tell both sides in real time; the swiper also gets the match in the HTTP response.
		if other, err := s.repo.GetMatch(ctx, req.UserID, outcome.MatchID); err == nil {
			s.realtime.Publish(req.UserID, realtime.Event{Type: "match.new", Data: other})
		}
		s.realtime.Publish(userID, realtime.Event{Type: "match.new", Data: match})
	}
	return result, nil
}

func (s *Service) ListMatches(ctx context.Context, userID string, before *time.Time, limit int) (MatchesResponse, error) {
	if limit <= 0 {
		limit = DefaultMatchesLimit
	}
	if limit > MaxMatchesLimit {
		limit = MaxMatchesLimit
	}
	matches, err := s.repo.ListMatches(ctx, userID, before, limit)
	if err != nil {
		return MatchesResponse{}, err
	}
	resp := MatchesResponse{Matches: matches}
	if len(matches) == limit {
		last := matches[len(matches)-1].CreatedAt
		resp.NextCursor = &last
	}
	return resp, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	match, err := s.repo.GetMatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.realtime.Publish(other, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID, "conversationId": match.ConversationID}})
	return nil
}
