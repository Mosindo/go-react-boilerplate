package matching

import (
	"context"
	"errors"
	"strings"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/httpx"
)

var (
	ErrInvalidAction = errors.New("action must be like or pass")
	ErrInvalidTarget = errors.New("invalid target")
)

// Events is told when a new match is formed (notifications, realtime). It is an
// interface so this package knows nothing about who listens.
type Events interface {
	MatchCreated(ctx context.Context, matchID, userA, userB string)
}

type ProfileSource interface {
	Cards(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
}

type Service struct {
	repo     Repository
	profiles ProfileSource
	events   Events
}

func NewService(repo Repository, source ProfileSource, events Events) *Service {
	return &Service{repo: repo, profiles: source, events: events}
}

func (s *Service) Swipe(ctx context.Context, actorID, targetID, action string) (SwipeResult, error) {
	if action != ActionLike && action != ActionPass {
		return SwipeResult{}, ErrInvalidAction
	}
	targetID = strings.ToLower(strings.TrimSpace(targetID))
	if !httpx.IsUUID(targetID) || targetID == actorID {
		return SwipeResult{}, ErrInvalidTarget
	}
	result, err := s.repo.Swipe(ctx, actorID, targetID, action)
	if err != nil {
		return SwipeResult{}, err
	}
	if result.Created {
		s.events.MatchCreated(ctx, result.MatchID, result.UserA, result.UserB)
	}
	return result, nil
}

func (s *Service) Matches(ctx context.Context, userID string, limit, offset int) ([]MatchResponse, error) {
	stored, err := s.repo.Matches(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(stored))
	for i, m := range stored {
		ids[i] = m.OtherID
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.PublicProfile, len(cards))
	for _, c := range cards {
		byID[c.ID] = c
	}
	out := make([]MatchResponse, 0, len(stored))
	for _, m := range stored {
		if card, ok := byID[m.OtherID]; ok {
			out = append(out, MatchResponse{ID: m.ID, CreatedAt: m.CreatedAt, Profile: card})
		}
	}
	return out, nil
}

func (s *Service) LikesReceived(ctx context.Context, userID string, limit, offset int) ([]profiles.PublicProfile, int, error) {
	ids, total, err := s.repo.LikesReceived(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	if len(ids) == 0 {
		return []profiles.PublicProfile{}, total, nil
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	return cards, total, err
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	return s.repo.Unmatch(ctx, userID, matchID)
}
