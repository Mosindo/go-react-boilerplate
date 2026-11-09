package profiles

import (
	"context"
	"time"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) Repo() Repository { return s.repo }

func (s *Service) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	return s.repo.GetOwn(ctx, userID)
}

func (s *Service) Save(ctx context.Context, userID string, in ProfileInput) (OwnProfile, error) {
	norm, birth, err := NormalizeProfile(in, s.now())
	if err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.Upsert(ctx, userID, norm, birth); err != nil {
		return OwnProfile{}, err
	}
	return s.repo.GetOwn(ctx, userID)
}

func (s *Service) SetLocation(ctx context.Context, userID string, in LocationInput) error {
	if in.Latitude == nil || in.Longitude == nil {
		return invalid("latitude and longitude are required")
	}
	if !validCoords(*in.Latitude, *in.Longitude) {
		return invalid("coordinates out of range")
	}
	return s.repo.SetLocation(ctx, userID, *in.Latitude, *in.Longitude)
}

func (s *Service) SavePreferences(ctx context.Context, userID string, p Preferences) (Preferences, error) {
	norm, err := NormalizePreferences(p)
	if err != nil {
		return Preferences{}, err
	}
	if _, err := s.repo.GetOwn(ctx, userID); err != nil {
		return Preferences{}, err
	}
	if err := s.repo.UpsertPreferences(ctx, userID, norm); err != nil {
		return Preferences{}, err
	}
	return norm, nil
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	return s.repo.GetPublic(ctx, viewerID, targetID)
}
