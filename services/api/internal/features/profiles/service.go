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

// Repo exposes the repository for features that need the shared visibility rule.
func (s *Service) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	return s.repo.CanView(ctx, viewerID, targetID)
}

func (s *Service) Catalog(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterestCatalog(ctx)
}

func (s *Service) Self(ctx context.Context, userID string) (SelfView, error) {
	view := SelfView{}
	missing := []string{}

	profile, err := s.repo.GetProfile(ctx, userID)
	switch {
	case err == nil:
		view.Profile = &profile
		if !profile.HasLocation {
			missing = append(missing, "location")
		}
	case errors.Is(err, ErrNotFound):
		missing = append(missing, "profile", "location")
	default:
		return SelfView{}, err
	}

	prefs, err := s.repo.GetPreferences(ctx, userID)
	switch {
	case err == nil:
		view.Preferences = &prefs
	case errors.Is(err, ErrNotFound):
		missing = append(missing, "preferences")
	default:
		return SelfView{}, err
	}

	interests, err := s.repo.GetInterests(ctx, []string{userID})
	if err != nil {
		return SelfView{}, err
	}
	view.Interests = interests[userID]

	photos, err := s.repo.GetPhotos(ctx, []string{userID})
	if err != nil {
		return SelfView{}, err
	}
	view.Photos = photos[userID]
	if len(view.Photos) == 0 {
		missing = append(missing, "photos")
	}

	view.Onboarding = Onboarding{Complete: len(missing) == 0, Missing: missing}
	return view, nil
}

func (s *Service) SaveProfile(ctx context.Context, userID string, req UpsertProfileRequest) error {
	name, err := normalizeFirstName(req.FirstName)
	if err != nil {
		return err
	}
	birthdate, err := parseBirthdate(req.Birthdate, s.now())
	if err != nil {
		return err
	}
	if _, ok := validGenders[req.Gender]; !ok {
		return ErrInvalidGender
	}
	bio, err := normalizeBio(req.Bio)
	if err != nil {
		return err
	}
	return s.repo.UpsertProfile(ctx, Profile{
		UserID: userID, FirstName: name, Birthdate: birthdate, Gender: req.Gender, Bio: bio,
	})
}

func (s *Service) SavePrivacy(ctx context.Context, userID string, showAge, showDistance, discoverable bool) error {
	return s.repo.UpdatePrivacy(ctx, userID, showAge, showDistance, discoverable)
}

func (s *Service) SavePreferences(ctx context.Context, userID string, req PreferencesRequest) error {
	prefs, err := normalizePreferences(Preferences{
		InterestedIn: req.InterestedIn, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm,
	})
	if err != nil {
		return err
	}
	return s.repo.UpsertPreferences(ctx, userID, prefs)
}

func (s *Service) SaveLocation(ctx context.Context, userID string, lat, lng float64, city string) error {
	lat, lng, city, err := normalizeLocation(lat, lng, city)
	if err != nil {
		return err
	}
	return s.repo.SetLocation(ctx, userID, lat, lng, city)
}

func (s *Service) SaveInterests(ctx context.Context, userID string, ids []int16) error {
	seen := make(map[int16]struct{}, len(ids))
	unique := make([]int16, 0, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) > MaxInterests {
		return ErrInvalidInterests
	}
	if err := s.repo.SetInterests(ctx, userID, unique); err != nil {
		if errors.Is(err, ErrUnknownInterests) {
			return ErrInvalidInterests
		}
		return err
	}
	return nil
}

// View returns another user's profile as the viewer may see it. Interests
// shared with the viewer are flagged.
func (s *Service) View(ctx context.Context, viewerID, targetID string) (PublicProfile, map[int16]struct{}, error) {
	pp, err := s.repo.GetViewable(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, nil, err
	}
	interests, err := s.repo.GetInterests(ctx, []string{viewerID, targetID})
	if err != nil {
		return PublicProfile{}, nil, err
	}
	photos, err := s.repo.GetPhotos(ctx, []string{targetID})
	if err != nil {
		return PublicProfile{}, nil, err
	}
	pp.Interests = interests[targetID]
	pp.Photos = photos[targetID]
	mine := make(map[int16]struct{})
	for _, i := range interests[viewerID] {
		mine[i.ID] = struct{}{}
	}
	shared := make(map[int16]struct{})
	for _, i := range pp.Interests {
		if _, ok := mine[i.ID]; ok {
			shared[i.ID] = struct{}{}
		}
	}
	return pp, shared, nil
}
