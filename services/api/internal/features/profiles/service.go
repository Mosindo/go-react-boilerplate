package profiles

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/features/photos"
	platformerrors "example.com/api/internal/platform/errors"
)

// PhotoLister is the part of the photos feature this service needs.
type PhotoLister interface {
	List(ctx context.Context, userID string) ([]photos.Photo, error)
}

type Service struct {
	repo   Repository
	photos PhotoLister
	now    func() time.Time
}

func NewService(repo Repository, photos PhotoLister) *Service {
	return &Service{repo: repo, photos: photos, now: time.Now}
}

func (s *Service) GetProfile(ctx context.Context, userID string) (Profile, error) {
	stored, err := s.repo.Get(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Profile{}, platformerrors.NotFound("profile_not_found", "profile not created yet")
		}
		return Profile{}, err
	}
	interests, err := s.repo.InterestsByUser(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	list, err := s.photos.List(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	if list == nil {
		list = []photos.Photo{}
	}
	return Profile{
		UserID:      stored.UserID,
		FirstName:   stored.FirstName,
		BirthDate:   stored.BirthDate.Format("2006-01-02"),
		Age:         AgeOn(stored.BirthDate, s.now()),
		Gender:      stored.Gender,
		Bio:         stored.Bio,
		City:        stored.City,
		HasLocation: stored.Latitude != nil,
		Interests:   interests,
		Photos:      list,
	}, nil
}

func (s *Service) SaveProfile(ctx context.Context, userID string, in ProfileInput) (Profile, error) {
	stored, slugs, err := normalizeProfile(userID, in, s.now())
	if err != nil {
		return Profile{}, err
	}
	if err := s.repo.Upsert(ctx, stored, in.ClearLocation, slugs); err != nil {
		if errors.Is(err, ErrRepositoryUnknownInterests) {
			return Profile{}, platformerrors.Validation(map[string]string{"interests": "unknown interest"})
		}
		return Profile{}, err
	}
	return s.GetProfile(ctx, userID)
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	p, err := s.repo.GetPreferences(ctx, userID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return Preferences{}, platformerrors.NotFound("preferences_not_found", "preferences not found")
	}
	return p, err
}

func (s *Service) SetPreferences(ctx context.Context, userID string, in Preferences) (Preferences, error) {
	p, err := validatePreferences(in)
	if err != nil {
		return Preferences{}, err
	}
	if err := s.repo.SetPreferences(ctx, userID, p); err != nil {
		return Preferences{}, err
	}
	return p, nil
}
