package profiles

import (
	"context"
	"errors"
	"time"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) Own(ctx context.Context, userID string) (OwnProfile, error) {
	return s.repo.GetOwn(ctx, userID)
}

// Save creates or updates the caller's profile. The birth date can be set once.
func (s *Service) Save(ctx context.Context, userID string, req ProfileRequest) (OwnProfile, error) {
	name, err := ValidateName(req.FirstName)
	if err != nil {
		return OwnProfile{}, err
	}
	birth, err := ValidateBirthDate(req.BirthDate, s.now())
	if err != nil {
		return OwnProfile{}, err
	}
	gender, err := ValidateGender(req.Gender)
	if err != nil {
		return OwnProfile{}, err
	}
	bio, err := ValidateBio(req.Bio)
	if err != nil {
		return OwnProfile{}, err
	}
	city, err := ValidateCity(req.City)
	if err != nil {
		return OwnProfile{}, err
	}
	interestIDs, err := ValidateInterestIDs(req.InterestIDs)
	if err != nil {
		return OwnProfile{}, err
	}

	input := ProfileInput{
		FirstName: name, BirthDate: birth, Gender: gender, Bio: bio, City: city,
		InterestIDs: interestIDs, IsVisible: true, ShowDistance: true,
	}
	existing, err := s.repo.GetOwn(ctx, userID)
	switch {
	case err == nil:
		if existing.BirthDate != birth.Format("2006-01-02") {
			return OwnProfile{}, &ValidationError{Message: "birth date cannot be changed"}
		}
		input.IsVisible, input.ShowDistance = existing.IsVisible, existing.ShowDistance
	case errors.Is(err, ErrNotFound):
	default:
		return OwnProfile{}, err
	}
	if req.IsVisible != nil {
		input.IsVisible = *req.IsVisible
	}
	if req.ShowDistance != nil {
		input.ShowDistance = *req.ShowDistance
	}

	if err := s.repo.UpsertProfile(ctx, userID, input); err != nil {
		if errors.Is(err, ErrUnknownInterest) {
			return OwnProfile{}, &ValidationError{Message: "unknown interest"}
		}
		return OwnProfile{}, err
	}
	return s.repo.GetOwn(ctx, userID)
}

func (s *Service) SavePreferences(ctx context.Context, userID string, req PreferencesRequest) (OwnProfile, error) {
	prefs, err := ValidatePreferences(req)
	if err != nil {
		return OwnProfile{}, err
	}
	if _, err := s.repo.GetOwn(ctx, userID); err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.UpsertPreferences(ctx, userID, prefs); err != nil {
		return OwnProfile{}, err
	}
	return s.repo.GetOwn(ctx, userID)
}

func (s *Service) SetLocation(ctx context.Context, userID string, req LocationRequest) (OwnProfile, error) {
	lat, lon, err := ValidateLocation(req)
	if err != nil {
		return OwnProfile{}, err
	}
	if err := s.repo.SetLocation(ctx, userID, lat, lon); err != nil {
		return OwnProfile{}, err
	}
	return s.repo.GetOwn(ctx, userID)
}

func (s *Service) Public(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	return s.repo.GetPublic(ctx, viewerID, targetID)
}

func (s *Service) PublicByIDs(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error) {
	return s.repo.PublicByIDs(ctx, viewerID, ids)
}

func (s *Service) CanView(ctx context.Context, viewerID, ownerID string) (bool, error) {
	return s.repo.CanView(ctx, viewerID, ownerID)
}
