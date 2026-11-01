package profiles

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/geo"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) SaveProfile(ctx context.Context, userID string, in ProfileInput) (Profile, error) {
	birth, err := in.Normalize(s.now())
	if err != nil {
		return Profile{}, err
	}
	if !geo.ValidLatLng(in.Latitude, in.Longitude) {
		return Profile{}, ErrInvalidInput
	}
	// Minimisation: seules des coordonnees arrondies (~1 km) sont conservees.
	in.Latitude = geo.RoundCoord(in.Latitude)
	in.Longitude = geo.RoundCoord(in.Longitude)
	if err := s.repo.UpsertProfile(ctx, userID, in, birth); err != nil {
		if errors.Is(err, ErrUnknownInterest) {
			return Profile{}, ErrInvalidInput
		}
		return Profile{}, err
	}
	return s.repo.GetProfile(ctx, userID)
}

func (s *Service) GetProfile(ctx context.Context, userID string) (Profile, error) {
	return s.repo.GetProfile(ctx, userID)
}

func (s *Service) GetSettings(ctx context.Context, userID string) (Settings, error) {
	return s.repo.GetSettings(ctx, userID)
}

func (s *Service) SaveSettings(ctx context.Context, userID string, in Settings) (Settings, error) {
	if err := in.Validate(); err != nil {
		return Settings{}, err
	}
	if err := s.repo.UpsertSettings(ctx, userID, in); err != nil {
		return Settings{}, err
	}
	return in, nil
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) PublicProfile(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	p, dist, showDist, err := s.repo.GetPublicProfile(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if showDist && viewerID != targetID {
		b := geo.BucketKm(dist)
		p.DistanceKm = &b
	}
	return p, nil
}
