package discover

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
)

var ErrIncompleteProfile = errors.New("complete your profile before discovering people")

// ProfileSource is the part of the profiles feature discovery depends on.
type ProfileSource interface {
	Own(ctx context.Context, userID string) (profiles.OwnProfile, error)
	PublicByIDs(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
}

type Service struct {
	repo     Repository
	profiles ProfileSource
	strategy Strategy
}

func NewService(repo Repository, source ProfileSource) *Service {
	return &Service{repo: repo, profiles: source, strategy: StrategyBalanced}
}

func (s *Service) requireComplete(ctx context.Context, userID string) error {
	own, err := s.profiles.Own(ctx, userID)
	if err != nil {
		if errors.Is(err, profiles.ErrNotFound) {
			return ErrIncompleteProfile
		}
		return err
	}
	if !own.IsComplete {
		return ErrIncompleteProfile
	}
	return nil
}

// Next returns the next batch of profiles for the viewer. Profiles disappear
// from the feed as soon as they are liked or passed, so the same call is
// naturally paginated by the viewer's own actions.
func (s *Service) Next(ctx context.Context, viewerID string, limit int) ([]profiles.PublicProfile, error) {
	if limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	if err := s.requireComplete(ctx, viewerID); err != nil {
		return nil, err
	}
	ids, err := s.repo.Candidates(ctx, viewerID, "", limit, s.strategy)
	if err != nil {
		return nil, err
	}
	return s.profiles.PublicByIDs(ctx, viewerID, ids)
}

// Eligible reports whether targetID may currently be liked or passed by viewerID.
func (s *Service) Eligible(ctx context.Context, viewerID, targetID string) (bool, error) {
	if viewerID == targetID {
		return false, nil
	}
	if err := s.requireComplete(ctx, viewerID); err != nil {
		if errors.Is(err, ErrIncompleteProfile) {
			return false, nil
		}
		return false, err
	}
	ids, err := s.repo.Candidates(ctx, viewerID, targetID, 1, s.strategy)
	if err != nil {
		return false, err
	}
	return len(ids) == 1, nil
}
