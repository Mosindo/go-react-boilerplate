package discover

import (
	"context"

	"example.com/api/internal/features/profiles"
)

type ProfileSource interface {
	Cards(ctx context.Context, viewerID string, ids []string) ([]profiles.PublicProfile, error)
	Touch(ctx context.Context, userID string) error
}

type Service struct {
	repo     Repository
	profiles ProfileSource
}

func NewService(repo Repository, source ProfileSource) *Service {
	return &Service{repo: repo, profiles: source}
}

// Next returns the next batch of profiles for the user. Profiles already swiped on
// never come back, so clients just call this again when their queue runs low.
func (s *Service) Next(ctx context.Context, userID string, limit int) ([]profiles.PublicProfile, error) {
	ids, err := s.repo.Candidates(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	_ = s.profiles.Touch(ctx, userID) // activity is a ranking hint, never worth failing a request
	if len(ids) == 0 {
		return []profiles.PublicProfile{}, nil
	}
	return s.profiles.Cards(ctx, userID, ids)
}
