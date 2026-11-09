package matching

import (
	"context"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

type Service struct {
	repo     Repository
	profiles profiles.Repository
	rt       realtime.Publisher
}

func NewService(repo Repository, profilesRepo profiles.Repository, rt realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: profilesRepo, rt: rt}
}

// Discover returns the next batch of profiles. Already-decided, blocked and matched
// profiles are excluded by the query, so the client never needs offsets.
func (s *Service) Discover(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	ids, err := s.repo.Candidates(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]profiles.PublicProfile, 0, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	cards, err := s.profiles.PublicBatch(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	for _, id := range ids { // keep ranking order
		if c, ok := cards[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResult, error) {
	if req.Action != ActionLike && req.Action != ActionPass {
		return SwipeResult{}, ErrInvalidAction
	}
	if req.TargetUserID == userID {
		return SwipeResult{}, ErrSelf
	}
	if !profiles.IsUUID(req.TargetUserID) {
		return SwipeResult{}, ErrTargetUnavailable
	}
	res, ev, err := s.repo.Swipe(ctx, userID, req.TargetUserID, req.Action)
	if err != nil {
		return SwipeResult{}, err
	}
	if ev != nil {
		for _, uid := range []string{ev.UserA, ev.UserB} {
			other := ev.UserA
			if uid == ev.UserA {
				other = ev.UserB
			}
			s.rt.Publish(uid, realtime.Event{Type: "match", Data: map[string]string{
				"matchId": ev.MatchID, "conversationId": ev.ConversationID, "userId": other,
			}})
		}
	}
	return res, nil
}

func (s *Service) ResetPasses(ctx context.Context, userID string) (int64, error) {
	return s.repo.ResetPasses(ctx, userID)
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	if !profiles.IsUUID(matchID) {
		return ErrMatchNotFound
	}
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.rt.Publish(other, realtime.Event{Type: "unmatch", Data: map[string]string{"matchId": matchID}})
	return nil
}
