package profiles

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/media"
)

var ErrNotVisible = errors.New("profile not visible")

type Service struct {
	repo   Repository
	signer *media.Signer
	now    func() time.Time
}

func NewService(repo Repository, signer *media.Signer) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) MyProfile(ctx context.Context, userID string) (MyProfile, error) {
	p, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return MyProfile{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return MyProfile{}, err
	}
	photos, err := s.repo.ListPhotos(ctx, []string{userID})
	if err != nil {
		return MyProfile{}, err
	}
	interests, err := s.repo.ListUserInterests(ctx, []string{userID})
	if err != nil {
		return MyProfile{}, err
	}

	out := MyProfile{
		UserID:      p.UserID,
		FirstName:   p.FirstName,
		Gender:      p.Gender,
		Bio:         p.Bio,
		JobTitle:    p.JobTitle,
		City:        p.City,
		HasLocation: p.Latitude != nil,
		Settings:    Settings{Discoverable: p.Discoverable, ShowDistance: p.ShowDistance},
		Preferences: prefs,
		Interests:   make([]Interest, 0, len(interests)),
		Photos:      s.mapPhotos(photos),
		IsDemo:      p.IsDemo,
		Missing:     MissingSteps(p, len(photos)),
	}
	if p.Birthdate != nil {
		formatted := p.Birthdate.Format("2006-01-02")
		age := AgeOn(*p.Birthdate, s.now())
		out.Birthdate = &formatted
		out.Age = &age
	}
	for _, i := range interests {
		out.Interests = append(out.Interests, i.Interest)
	}
	out.IsComplete = len(out.Missing) == 0
	return out, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (MyProfile, error) {
	patch := ProfilePatch{}
	if req.FirstName != nil {
		name, err := NormalizeFirstName(*req.FirstName)
		if err != nil {
			return MyProfile{}, err
		}
		patch.FirstName = &name
	}
	if req.Birthdate != nil {
		birth, err := ParseBirthdate(*req.Birthdate, s.now())
		if err != nil {
			return MyProfile{}, err
		}
		patch.Birthdate = &birth
	}
	if req.Gender != nil {
		if !ValidGender(*req.Gender) {
			return MyProfile{}, ErrInvalidGender
		}
		patch.Gender = req.Gender
	}
	if req.Bio != nil {
		bio, err := NormalizeFreeText(*req.Bio, MaxBioRunes, ErrBioTooLong)
		if err != nil {
			return MyProfile{}, err
		}
		patch.Bio = &bio
	}
	if req.JobTitle != nil {
		job, err := NormalizeFreeText(*req.JobTitle, MaxJobTitleRunes, ErrJobTitleTooLong)
		if err != nil {
			return MyProfile{}, err
		}
		patch.JobTitle = &job
	}
	if req.InterestIDs != nil {
		ids := dedupeInts(*req.InterestIDs)
		if len(ids) > MaxInterests {
			return MyProfile{}, ErrTooManyInterests
		}
		if len(ids) > 0 {
			known, err := s.repo.CountKnownInterests(ctx, ids)
			if err != nil {
				return MyProfile{}, err
			}
			if known != len(ids) {
				return MyProfile{}, ErrUnknownInterest
			}
		}
		patch.InterestIDs = &ids
	}
	if err := s.repo.UpdateProfile(ctx, userID, patch); err != nil {
		return MyProfile{}, err
	}
	return s.MyProfile(ctx, userID)
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, lat, lng float64, city *string) (MyProfile, error) {
	if err := ValidateCoordinates(lat, lng); err != nil {
		return MyProfile{}, err
	}
	if city != nil {
		normalized, err := NormalizeFreeText(*city, MaxCityRunes, ErrCityTooLong)
		if err != nil {
			return MyProfile{}, err
		}
		city = &normalized
	}
	if err := s.repo.UpdateLocation(ctx, userID, RoundCoordinate(lat), RoundCoordinate(lng), city); err != nil {
		return MyProfile{}, err
	}
	return s.MyProfile(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (MyProfile, error) {
	prefs, err := ValidatePreferences(req)
	if err != nil {
		return MyProfile{}, err
	}
	if err := s.repo.UpdatePreferences(ctx, userID, prefs); err != nil {
		return MyProfile{}, err
	}
	return s.MyProfile(ctx, userID)
}

func (s *Service) UpdateSettings(ctx context.Context, userID string, req UpdateSettingsRequest) (MyProfile, error) {
	if err := s.repo.UpdateSettings(ctx, userID, req.Discoverable, req.ShowDistance); err != nil {
		return MyProfile{}, err
	}
	return s.MyProfile(ctx, userID)
}

// ViewProfile returns another member's public profile if the viewer may see it.
func (s *Service) ViewProfile(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	if viewerID == targetID {
		return PublicProfile{}, ErrNotVisible
	}
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if !ok {
		return PublicProfile{}, ErrNotVisible
	}
	cards, err := s.PublicProfiles(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	if len(cards) == 0 {
		return PublicProfile{}, ErrNotVisible
	}
	return cards[0], nil
}

// PublicProfiles loads cards for several users in three queries (no N+1), preserving
// the order of userIDs. Callers are responsible for authorisation.
func (s *Service) PublicProfiles(ctx context.Context, viewerID string, userIDs []string) ([]PublicProfile, error) {
	if len(userIDs) == 0 {
		return []PublicProfile{}, nil
	}
	viewer, err := s.repo.GetProfile(ctx, viewerID)
	if err != nil {
		return nil, err
	}
	stored, err := s.repo.GetProfiles(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.ListPhotos(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.ListUserInterests(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	photosByUser := map[string][]StoredPhoto{}
	for _, p := range photos {
		photosByUser[p.UserID] = append(photosByUser[p.UserID], p)
	}
	interestsByUser := map[string][]Interest{}
	for _, i := range interests {
		interestsByUser[i.UserID] = append(interestsByUser[i.UserID], i.Interest)
	}
	byID := make(map[string]StoredProfile, len(stored))
	for _, p := range stored {
		byID[p.UserID] = p
	}

	now := s.now()
	out := make([]PublicProfile, 0, len(userIDs))
	for _, id := range userIDs {
		p, ok := byID[id]
		if !ok || p.Birthdate == nil || p.Gender == nil {
			continue
		}
		card := PublicProfile{
			UserID:    p.UserID,
			FirstName: p.FirstName,
			Age:       AgeOn(*p.Birthdate, now),
			Gender:    *p.Gender,
			Bio:       p.Bio,
			JobTitle:  p.JobTitle,
			City:      p.City,
			Interests: interestsByUser[id],
			Photos:    s.mapPhotos(photosByUser[id]),
			IsDemo:    p.IsDemo,
		}
		if card.Interests == nil {
			card.Interests = []Interest{}
		}
		if p.ShowDistance && p.Latitude != nil && viewer.Latitude != nil {
			km := ApproximateDistanceKm(HaversineKm(*viewer.Latitude, *viewer.Longitude, *p.Latitude, *p.Longitude))
			card.DistanceKm = &km
		}
		out = append(out, card)
	}
	return out, nil
}

// RequireComplete is used by features (discovery, swipes) that need a finished profile.
func (s *Service) RequireComplete(ctx context.Context, userID string) error {
	p, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return err
	}
	photos, err := s.repo.ListPhotos(ctx, []string{userID})
	if err != nil {
		return err
	}
	if len(MissingSteps(p, len(photos))) > 0 {
		return ErrProfileIncomplete
	}
	return nil
}

func (s *Service) mapPhotos(photos []StoredPhoto) []Photo {
	out := make([]Photo, 0, len(photos))
	for _, p := range photos {
		out = append(out, Photo{ID: p.ID, URL: s.signer.PhotoURL(p.ID), Position: p.Position, Width: p.Width, Height: p.Height})
	}
	return out
}

func dedupeInts(values []int) []int {
	seen := make(map[int]bool, len(values))
	out := make([]int, 0, len(values))
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

// ViewerLocation exposes the stored (already rounded) coordinates to server-side
// features only; they are never serialised to clients.
func (s *Service) ViewerLocation(ctx context.Context, userID string) (float64, float64, bool, error) {
	p, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return 0, 0, false, err
	}
	if p.Latitude == nil || p.Longitude == nil {
		return 0, 0, false, nil
	}
	return *p.Latitude, *p.Longitude, true, nil
}
