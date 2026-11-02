package profiles

import (
	"context"
	"errors"
	"time"

	perr "example.com/api/internal/platform/errors"
)

type Service struct {
	repo *Repository
	now  func() time.Time
}

func NewService(repo *Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (Bundle, error) {
	v, err := validateProfile(req, s.now())
	if err != nil {
		return Bundle{}, err
	}
	if err := s.repo.Upsert(ctx, userID, v); err != nil {
		if errors.Is(err, ErrUnknownInterest) {
			return Bundle{}, perr.BadRequest("invalid_interests", "Centre d'intérêt inconnu.")
		}
		return Bundle{}, err
	}
	return s.Get(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (Bundle, error) {
	p, err := validatePreferences(req)
	if err != nil {
		return Bundle{}, err
	}
	if err := s.repo.UpsertPreferences(ctx, userID, p); err != nil {
		return Bundle{}, err
	}
	return s.Get(ctx, userID)
}

func (s *Service) Get(ctx context.Context, userID string) (Bundle, error) {
	p, prefs, err := s.repo.Get(ctx, userID)
	if err != nil {
		return Bundle{}, err
	}
	b := Bundle{Profile: p, Preferences: prefs}
	b.Complete = p != nil && prefs != nil && len(p.Photos) > 0 && p.Latitude != nil
	return b, nil
}
