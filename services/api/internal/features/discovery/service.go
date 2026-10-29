package discovery

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo      Repository
	publisher realtime.Publisher
}

func NewService(repo Repository, publisher realtime.Publisher) *Service {
	return &Service{repo: repo, publisher: publisher}
}

func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]*profiles.PublicProfile, error) {
	items, err := s.repo.Candidates(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*profiles.PublicProfile{}
	}
	return items, nil
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResponse, error) {
	if req.Action != ActionLike && req.Action != ActionPass {
		return SwipeResponse{}, platformerrors.Invalid("action must be like or pass")
	}
	if !profiles.IsUUID(req.UserID) {
		return SwipeResponse{}, platformerrors.Invalid("invalid user id")
	}
	if req.UserID == userID {
		return SwipeResponse{}, platformerrors.Invalid("you cannot swipe on yourself")
	}

	res, err := s.repo.Swipe(ctx, userID, req.UserID, req.Action)
	switch {
	case errors.Is(err, ErrNotOnboarded):
		return SwipeResponse{}, platformerrors.Forbidden("complete your profile before discovering people")
	case errors.Is(err, ErrTargetUnavailable):
		return SwipeResponse{}, platformerrors.NotFound("profile not available")
	case errors.Is(err, ErrConflictingSwipe):
		return SwipeResponse{}, platformerrors.Conflict("you already answered this profile")
	case err != nil:
		return SwipeResponse{}, err
	}

	resp := SwipeResponse{Action: res.Action, AlreadyProcessed: res.AlreadyProcessed, Matched: res.Matched}
	if res.Matched {
		card := res.Cards[userID]
		resp.Match = &Match{ID: res.MatchID, ConversationID: res.ConversationID, User: card}
	}

	if res.Matched && !res.AlreadyProcessed {
		for uid, card := range res.Cards {
			s.publisher.Publish(uid, realtime.Event{Type: "match", Data: Match{ID: res.MatchID, ConversationID: res.ConversationID, User: card}})
			if n, ok := res.Notifications[uid]; ok {
				s.publisher.Publish(uid, realtime.Event{Type: "notification", Data: n})
			}
		}
	}
	return resp, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	if !profiles.IsUUID(matchID) {
		return platformerrors.NotFound("match not found")
	}
	u, err := s.repo.Unmatch(ctx, userID, matchID)
	if errors.Is(err, ErrMatchNotFound) {
		return platformerrors.NotFound("match not found")
	}
	if err != nil {
		return err
	}
	s.publisher.Publish(u.OtherUserID, realtime.Event{Type: "unmatch", Data: map[string]string{"matchId": u.MatchID, "conversationId": u.ConversationID}})
	return nil
}
