package matches

import (
	"context"
	"time"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

// Eligibility is implemented by the discover feature: the single source of
// truth for who a user may like or pass.
type Eligibility interface {
	Eligible(ctx context.Context, viewerID, targetID string) (bool, error)
}

type ProfileSource interface {
	PublicByIDs(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
}

type Service struct {
	repo     Repository
	elig     Eligibility
	profiles ProfileSource
	events   realtime.Publisher
}

func NewService(repo Repository, elig Eligibility, source ProfileSource, events realtime.Publisher) *Service {
	if events == nil {
		events = realtime.Discard{}
	}
	return &Service{repo: repo, elig: elig, profiles: source, events: events}
}

// Swipe records a like or a pass. A like answered by an earlier like from the
// other person creates the match, its conversation and both notifications.
func (s *Service) Swipe(ctx context.Context, userID, targetID, action string) (LikeResult, error) {
	acted, err := s.repo.HasActed(ctx, userID, targetID)
	if err != nil {
		return LikeResult{}, err
	}
	if acted {
		return LikeResult{}, ErrAlreadyActed
	}
	ok, err := s.elig.Eligible(ctx, userID, targetID)
	if err != nil {
		return LikeResult{}, err
	}
	if !ok {
		return LikeResult{}, ErrNotEligible
	}
	res, err := s.repo.Swipe(ctx, userID, targetID, action)
	if err != nil {
		return LikeResult{}, err
	}
	if res.Matched && res.Match != nil {
		for _, pair := range [][2]string{{userID, targetID}, {targetID, userID}} {
			s.events.Publish(pair[0], realtime.Event{Type: "match", Data: map[string]string{
				"matchId": res.Match.ID, "conversationId": res.Match.ConversationID, "userId": pair[1],
			}})
		}
	}
	return res, nil
}

func (s *Service) List(ctx context.Context, userID string, before *time.Time, limit int) (MatchesResponse, error) {
	if limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	stored, err := s.repo.List(ctx, userID, before, limit)
	if err != nil {
		return MatchesResponse{}, err
	}
	ids := make([]string, len(stored))
	for i, m := range stored {
		ids[i] = m.OtherUserID
	}
	people, err := s.profiles.PublicByIDs(ctx, userID, ids)
	if err != nil {
		return MatchesResponse{}, err
	}
	byID := make(map[string]profiles.PublicProfile, len(people))
	for _, p := range people {
		byID[p.ID] = p
	}
	resp := MatchesResponse{Matches: make([]Match, 0, len(stored))}
	for _, m := range stored {
		p, ok := byID[m.OtherUserID]
		if !ok {
			continue
		}
		resp.Matches = append(resp.Matches, Match{
			ID: m.ID, ConversationID: m.ConversationID, CreatedAt: m.CreatedAt, LastMessageAt: m.LastMessageAt, Profile: p,
		})
	}
	if len(stored) == limit {
		last := stored[len(stored)-1].CreatedAt
		resp.NextBefore = &last
	}
	return resp, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, userID, matchID)
	if err != nil {
		return err
	}
	s.events.Publish(other, realtime.Event{Type: "unmatch", Data: map[string]string{"matchId": matchID}})
	return nil
}
