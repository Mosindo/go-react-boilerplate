package profiles

import (
	"context"
	"errors"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
)

var ErrUnknownInterest = platformerrors.BadRequest("unknown_interest", "unknown interest")

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) Me(ctx context.Context, userID string) (MeResponse, error) {
	account, err := s.repo.GetAccount(ctx, userID)
	if err != nil {
		return MeResponse{}, platformerrors.Unauthorized("unauthorized", "user not found")
	}
	stored, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return MeResponse{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return MeResponse{}, err
	}
	photos, err := s.repo.ListPhotos(ctx, userID)
	if err != nil {
		return MeResponse{}, err
	}
	res := MeResponse{
		User: account, Preferences: prefs, Photos: photos,
		Onboarding: Onboarding{HasProfile: stored != nil, HasPhoto: len(photos) > 0},
	}
	res.Onboarding.Complete = res.Onboarding.HasProfile && res.Onboarding.HasPhoto
	if stored != nil {
		interests, err := s.repo.GetInterests(ctx, userID)
		if err != nil {
			return MeResponse{}, err
		}
		res.Profile = &OwnProfile{
			FirstName: stored.FirstName, Birthdate: stored.Birthdate.Format("2006-01-02"), Age: AgeOn(stored.Birthdate, s.now()),
			Gender: stored.Gender, Bio: stored.Bio, Occupation: stored.Occupation, LookingFor: stored.LookingFor,
			City: stored.City, HasLocation: stored.HasLocation, Interests: interests,
		}
	}
	return res, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (MeResponse, error) {
	v, err := validateProfile(req, s.now())
	if err != nil {
		return MeResponse{}, err
	}
	if err := s.repo.UpsertProfile(ctx, userID, v); err != nil {
		if errors.Is(err, ErrUnknownInterest) {
			return MeResponse{}, ErrUnknownInterest
		}
		return MeResponse{}, err
	}
	return s.Me(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (MeResponse, error) {
	p, err := validatePreferences(req)
	if err != nil {
		return MeResponse{}, err
	}
	if err := s.repo.UpsertPreferences(ctx, userID, p); err != nil {
		return MeResponse{}, err
	}
	return s.Me(ctx, userID)
}

func (s *Service) SetLocation(ctx context.Context, userID string, req UpdateLocationRequest) (MeResponse, error) {
	lat, lng, city, err := validateLocation(req)
	if err != nil {
		return MeResponse{}, err
	}
	if err := s.repo.SetLocation(ctx, userID, lat, lng, city); err != nil {
		if errors.Is(err, ErrNoProfile) {
			return MeResponse{}, platformerrors.Conflict("profile_required", "create your profile first")
		}
		return MeResponse{}, err
	}
	return s.Me(ctx, userID)
}

func (s *Service) ClearLocation(ctx context.Context, userID string) (MeResponse, error) {
	if err := s.repo.ClearLocation(ctx, userID); err != nil {
		return MeResponse{}, err
	}
	return s.Me(ctx, userID)
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// View returns another user's public profile, or 404 when the viewer may not see it
// (unknown, suspended, hidden, or blocked in either direction).
func (s *Service) View(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	notFound := platformerrors.NotFound("not_found", "profile not found")
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if !ok {
		return PublicProfile{}, notFound
	}
	cards, err := s.repo.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	card, found := cards[targetID]
	if !found {
		return PublicProfile{}, notFound
	}
	return card, nil
}

// Cards exposes batch loading to other features (discovery, chat).
func (s *Service) Cards(ctx context.Context, viewerID string, ids []string) (map[string]PublicProfile, error) {
	return s.repo.Cards(ctx, viewerID, ids)
}

// CanView reports whether viewer may see target's profile and photos.
func (s *Service) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	return s.repo.CanView(ctx, viewerID, targetID)
}
