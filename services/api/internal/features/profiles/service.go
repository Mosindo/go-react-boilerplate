package profiles

import (
	"context"
	"errors"

	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/media"
)

var ErrForbidden = errors.New("profile not visible")

type Service struct {
	repo   Repository
	signer *media.Signer
}

func NewService(repo Repository, signer *media.Signer) *Service {
	return &Service{repo: repo, signer: signer}
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) Own(ctx context.Context, userID string) (OwnProfile, error) {
	p, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}
	interests, err := s.repo.InterestsFor(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	photos, err := s.repo.PhotosFor(ctx, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	out := OwnProfile{
		UserID:      p.UserID,
		FirstName:   p.FirstName,
		Birthdate:   p.Birthdate.Format("2006-01-02"),
		Age:         AgeAt(p.Birthdate, timeNow()),
		Gender:      p.Gender,
		Bio:         p.Bio,
		JobTitle:    p.JobTitle,
		LookingFor:  p.LookingFor,
		City:        p.City,
		HasLocation: p.Latitude != nil,
		Interests:   nonNil(interests[userID]),
		Photos:      s.photoViews(photos[userID]),
		Preferences: prefs,
	}
	out.IsComplete = len(out.Photos) > 0
	return out, nil
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (OwnProfile, error) {
	normalized, err := normalizeProfile(req, timeNow())
	if err != nil {
		return OwnProfile{}, err
	}
	if existing, err := s.repo.GetProfile(ctx, userID); err == nil {
		// The birthdate is immutable once set; ignore any other value silently only
		// if it matches, otherwise reject so clients cannot believe it was changed.
		if !existing.Birthdate.Equal(normalized.Birthdate) {
			return OwnProfile{}, ErrBirthdateLocked
		}
	} else if !errors.Is(err, ErrNotFound) {
		return OwnProfile{}, err
	}
	if err := s.repo.UpsertProfile(ctx, userID, normalized); err != nil {
		return OwnProfile{}, err
	}
	return s.Own(ctx, userID)
}

var ErrBirthdateLocked = errors.New("birthdate cannot be changed after profile creation")

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (Preferences, error) {
	prefs, err := normalizePreferences(req)
	if err != nil {
		return Preferences{}, err
	}
	if err := s.repo.UpdatePreferences(ctx, userID, prefs); err != nil {
		return Preferences{}, err
	}
	return prefs, nil
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, req UpdateLocationRequest) error {
	lat, lng := *req.Latitude, *req.Longitude
	if !geo.ValidCoordinates(lat, lng) {
		return ErrInvalidLocation
	}
	var city *string
	if req.City != nil {
		c := cleanText(*req.City, false)
		if len([]rune(c)) > 80 {
			return ErrInvalidCity
		}
		city = &c
	}
	return s.repo.UpdateLocation(ctx, userID, geo.Round(lat), geo.Round(lng), city)
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	return s.repo.ClearLocation(ctx, userID)
}

// View returns another user's card if the viewer is allowed to see it.
func (s *Service) View(ctx context.Context, viewerID, targetID string) (Card, error) {
	if viewerID != targetID {
		ok, err := s.repo.CanView(ctx, viewerID, targetID)
		if err != nil {
			return Card{}, err
		}
		if !ok {
			return Card{}, ErrForbidden
		}
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return Card{}, err
	}
	card, ok := cards[targetID]
	if !ok {
		return Card{}, ErrNotFound
	}
	return card, nil
}

// Cards loads public cards for many users with a constant number of queries.
// Authorisation is the caller's responsibility.
func (s *Service) Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]Card, error) {
	out := make(map[string]Card, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := s.repo.CardRows(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.InterestsFor(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.PhotosFor(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	var viewerLat, viewerLng *float64
	if viewer, err := s.repo.GetProfile(ctx, viewerID); err == nil {
		viewerLat, viewerLng = viewer.Latitude, viewer.Longitude
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	now := timeNow()
	for _, row := range rows {
		card := Card{
			UserID:     row.UserID,
			FirstName:  row.FirstName,
			Age:        AgeAt(row.Birthdate, now),
			Gender:     row.Gender,
			Bio:        row.Bio,
			JobTitle:   row.JobTitle,
			LookingFor: row.LookingFor,
			City:       row.City,
			Interests:  nonNil(interests[row.UserID]),
			Photos:     s.photoViews(photos[row.UserID]),
		}
		if row.ShowDistance && viewerLat != nil && row.Latitude != nil {
			d := geo.ApproximateKm(geo.DistanceKm(*viewerLat, *viewerLng, *row.Latitude, *row.Longitude))
			card.DistanceKm = &d
		}
		out[row.UserID] = card
	}
	return out, nil
}

func (s *Service) photoViews(stored []StoredPhoto) []Photo {
	out := make([]Photo, 0, len(stored))
	for _, p := range stored {
		out = append(out, Photo{ID: p.ID, URL: s.signer.URL(p.StorageKey), Position: p.Position, Width: p.Width, Height: p.Height})
	}
	return out
}

func nonNil[T any](v []T) []T {
	if v == nil {
		return []T{}
	}
	return v
}
