package discovery

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
)

const (
	DefaultLimit = 10
	MaxLimit     = 20
)

var ErrOnboardingIncomplete = errors.New("complete your profile, photos and preferences first")

// Profiles is the subset of the profiles service discovery depends on.
type Profiles interface {
	IsOnboarded(ctx context.Context, userID string) (bool, error)
	Cards(ctx context.Context, viewerID string, userIDs []string) ([]profiles.Card, error)
}

type Service struct {
	repo     Repository
	profiles Profiles
}

func NewService(repo Repository, p Profiles) *Service { return &Service{repo: repo, profiles: p} }

func normalizeLimit(n int) int {
	if n <= 0 {
		return DefaultLimit
	}
	if n > MaxLimit {
		return MaxLimit
	}
	return n
}

// Next returns the next batch of profiles. Profiles disappear from the feed as
// soon as the viewer likes or passes them, so no cursor is needed.
func (s *Service) Next(ctx context.Context, viewerID string, limit int) ([]profiles.Card, error) {
	ok, err := s.profiles.IsOnboarded(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrOnboardingIncomplete
	}
	ids, err := s.repo.Candidates(ctx, viewerID, normalizeLimit(limit))
	if err != nil {
		return nil, err
	}
	return s.profiles.Cards(ctx, viewerID, ids)
}
