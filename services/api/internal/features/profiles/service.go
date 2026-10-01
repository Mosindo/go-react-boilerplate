package profiles

import (
	"context"
	"strings"
	"time"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) Get(ctx context.Context, userID string) (SelfProfile, error) {
	return s.repo.GetSelf(ctx, userID)
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (SelfProfile, error) {
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return SelfProfile{}, ErrInvalidBirthDate
	}
	in := ProfileInput{
		FirstName:    CleanText(req.FirstName),
		BirthDate:    birth,
		Gender:       strings.TrimSpace(req.Gender),
		Bio:          CleanText(req.Bio),
		City:         CleanText(req.City),
		IsVisible:    req.IsVisible,
		ShowDistance: req.ShowDistance,
	}
	if err := ValidateProfileText(in.FirstName, in.Bio, in.City); err != nil {
		return SelfProfile{}, err
	}
	if err := ValidateBirthDate(birth, time.Now()); err != nil {
		return SelfProfile{}, err
	}
	if !ValidGender(in.Gender) {
		return SelfProfile{}, ErrInvalidGender
	}
	if err := s.repo.Upsert(ctx, userID, in); err != nil {
		return SelfProfile{}, err
	}
	return s.repo.GetSelf(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req PreferencesRequest) (SelfProfile, error) {
	p := Preferences{InterestedIn: req.InterestedIn, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}
	if err := ValidatePreferences(p); err != nil {
		return SelfProfile{}, err
	}
	if err := s.repo.UpdatePreferences(ctx, userID, p); err != nil {
		return SelfProfile{}, err
	}
	return s.repo.GetSelf(ctx, userID)
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, req LocationRequest) (SelfProfile, error) {
	if err := ValidateLocation(req.Latitude, req.Longitude); err != nil {
		return SelfProfile{}, err
	}
	city := CleanText(req.City)
	if err := ValidateProfileText("x", "", city); err != nil {
		return SelfProfile{}, err
	}
	if err := s.repo.UpdateLocation(ctx, userID, SnapCoordinate(req.Latitude), SnapCoordinate(req.Longitude), city); err != nil {
		return SelfProfile{}, err
	}
	return s.repo.GetSelf(ctx, userID)
}

func (s *Service) SetInterests(ctx context.Context, userID string, ids []int) (SelfProfile, error) {
	seen := map[int]bool{}
	for _, id := range ids {
		if id <= 0 || id > 32767 || seen[id] {
			return SelfProfile{}, ErrInvalidInterests
		}
		seen[id] = true
	}
	if len(ids) > MaxInterests {
		return SelfProfile{}, ErrInvalidInterests
	}
	if err := s.repo.SetInterests(ctx, userID, ids); err != nil {
		return SelfProfile{}, err
	}
	return s.repo.GetSelf(ctx, userID)
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// Cards exposes batched public cards to other features (discovery, matches, chat).
func (s *Service) Cards(ctx context.Context, viewerID string, ids []string) (map[string]Card, error) {
	return s.repo.CardsByIDs(ctx, viewerID, ids)
}
