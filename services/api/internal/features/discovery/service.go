package discovery

import (
	"context"
	"errors"

	"example.com/api/internal/features/profiles"
	perr "example.com/api/internal/platform/errors"
)

// ProfileLoader builds public profiles for ids the caller may see (satisfied by profiles.Service).
type ProfileLoader interface {
	LoadPublic(ctx context.Context, viewerID string, ids []string, exactDistances map[string]float64) ([]profiles.PublicProfile, error)
}

type Service struct {
	repo     Repository
	profiles ProfileLoader
}

func NewService(repo Repository, loader ProfileLoader) *Service {
	return &Service{repo: repo, profiles: loader}
}

func (s *Service) Next(ctx context.Context, userID string, limit int) (Page, error) {
	cands, err := s.repo.Candidates(ctx, userID, limit)
	if err != nil {
		if errors.Is(err, ErrNoProfile) {
			return Page{}, perr.Conflict("profile_incomplete", "create your profile to discover people")
		}
		return Page{}, err
	}
	ids := make([]string, 0, len(cands))
	distances := make(map[string]float64, len(cands))
	for _, c := range cands {
		ids = append(ids, c.UserID)
		if c.DistanceKm != nil {
			distances[c.UserID] = *c.DistanceKm
		}
	}
	items, err := s.profiles.LoadPublic(ctx, userID, ids, distances)
	if err != nil {
		return Page{}, err
	}
	if items == nil {
		items = []profiles.PublicProfile{}
	}
	return Page{Items: items}, nil
}
