package matching

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/realtime"
)

var (
	ErrValidation           = errors.New("invalid swipe")
	ErrOnboardingIncomplete = errors.New("complete your profile, photos and preferences first")
	ErrTargetUnavailable    = errors.New("profile not available")
	ErrNotFound             = errors.New("match not found")
)

const (
	DefaultLimit = 30
	MaxLimit     = 50
	MaxOffset    = 1000
)

// Profiles is the subset of the profiles service matching depends on.
type Profiles interface {
	IsOnboarded(ctx context.Context, userID string) (bool, error)
	CanView(ctx context.Context, viewerID, targetID string) (bool, error)
	Cards(ctx context.Context, viewerID string, userIDs []string) ([]profiles.Card, error)
}

type Service struct {
	repo     Repository
	profiles Profiles
	events   realtime.Publisher
}

func NewService(repo Repository, p Profiles, events realtime.Publisher) *Service {
	return &Service{repo: repo, profiles: p, events: events}
}

// Swipe records a like or a pass. All business rules are enforced here, never
// trusted from the client: onboarding, self-swipes, visibility, blocks, duplicates.
func (s *Service) Swipe(ctx context.Context, userID string, req SwipeRequest) (SwipeResult, error) {
	if req.Action != "like" && req.Action != "pass" {
		return SwipeResult{}, ErrValidation
	}
	if req.TargetID == userID {
		return SwipeResult{}, ErrValidation
	}
	ok, err := s.profiles.IsOnboarded(ctx, userID)
	if err != nil {
		return SwipeResult{}, err
	}
	if !ok {
		return SwipeResult{}, ErrOnboardingIncomplete
	}
	visible, err := s.profiles.CanView(ctx, userID, req.TargetID)
	if err != nil {
		return SwipeResult{}, err
	}
	if !visible {
		return SwipeResult{}, ErrTargetUnavailable
	}

	out, err := s.repo.Swipe(ctx, userID, req.TargetID, req.Action)
	if err != nil {
		return SwipeResult{}, err
	}
	res := SwipeResult{AlreadyProcessed: out.AlreadyProcessed}
	if out.MatchID != "" {
		res.Matched = true
		res.MatchID = &out.MatchID
		if !out.AlreadyProcessed {
			for _, uid := range []string{userID, req.TargetID} {
				s.events.Publish(uid, realtime.Event{Type: "match.new", Data: map[string]string{"matchId": out.MatchID}})
				s.events.Publish(uid, realtime.Event{Type: "notification.new"})
			}
		}
	}
	return res, nil
}

func normalizePage(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if offset < 0 {
		offset = 0
	}
	if offset > MaxOffset {
		offset = MaxOffset
	}
	return limit, offset
}

func (s *Service) List(ctx context.Context, userID string, limit, offset int) ([]MatchSummary, error) {
	limit, offset = normalizePage(limit, offset)
	stored, err := s.repo.ListMatches(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(stored))
	for _, m := range stored {
		ids = append(ids, m.OtherUserID)
	}
	cards, err := s.profiles.Cards(ctx, userID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]profiles.Card, len(cards))
	for _, c := range cards {
		byID[c.UserID] = c
	}
	out := make([]MatchSummary, 0, len(stored))
	for _, m := range stored {
		card, ok := byID[m.OtherUserID]
		if !ok {
			continue
		}
		out = append(out, MatchSummary{
			ID: m.ID, CreatedAt: m.CreatedAt, LastActivityAt: m.LastActivityAt,
			Profile: card, LastMessage: m.LastMessage, UnreadCount: m.UnreadCount,
		})
	}
	return out, nil
}

func (s *Service) Unmatch(ctx context.Context, userID, matchID string) error {
	other, err := s.repo.Unmatch(ctx, matchID, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrNotFound
		}
		return err
	}
	s.events.Publish(other, realtime.Event{Type: "match.removed", Data: map[string]string{"matchId": matchID}})
	return nil
}
