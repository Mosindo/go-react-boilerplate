package profiles

import (
	"context"
	"errors"
	"time"

	"example.com/api/internal/platform/media"
)

var ErrForbidden = errors.New("profile not visible")

type Service struct {
	repo   Repository
	signer *media.Signer
	now    func() time.Time
}

func NewService(repo Repository, signer *media.Signer) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) Me(ctx context.Context, userID string) (MeResponse, error) {
	var resp MeResponse
	stored, err := s.repo.GetProfile(ctx, userID)
	switch {
	case errors.Is(err, ErrNotFound):
	case err != nil:
		return resp, err
	default:
		interests, err := s.repo.ListUserInterests(ctx, userID)
		if err != nil {
			return resp, err
		}
		photos, err := s.repo.LoadPhotos(ctx, []string{userID})
		if err != nil {
			return resp, err
		}
		resp.Profile = &MyProfile{
			FirstName:      stored.FirstName,
			Birthdate:      stored.Birthdate.Format("2006-01-02"),
			Age:            AgeOn(stored.Birthdate, s.now()),
			Gender:         stored.Gender,
			Bio:            stored.Bio,
			JobTitle:       stored.JobTitle,
			HeightCm:       stored.HeightCm,
			LookingFor:     stored.LookingFor,
			City:           stored.City,
			HasLocation:    stored.Latitude != nil,
			IsDiscoverable: stored.IsDiscoverable,
			Interests:      nonNil(interests),
			Photos:         s.mapPhotos(photos),
		}
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	switch {
	case errors.Is(err, ErrNotFound):
	case err != nil:
		return resp, err
	default:
		resp.Preferences = &prefs
	}
	resp.Completeness = Completeness{
		HasProfile:     resp.Profile != nil,
		HasPreferences: resp.Preferences != nil,
		HasPhotos:      resp.Profile != nil && len(resp.Profile.Photos) > 0,
		HasLocation:    resp.Profile != nil && resp.Profile.HasLocation,
	}
	resp.Completeness.IsComplete = resp.Completeness.HasProfile && resp.Completeness.HasPreferences && resp.Completeness.HasPhotos
	return resp, nil
}

func (s *Service) SaveProfile(ctx context.Context, userID string, in ProfileInput) (MeResponse, error) {
	rec, err := validateProfile(in, s.now())
	if err != nil {
		return MeResponse{}, err
	}
	if err := s.repo.UpsertProfile(ctx, userID, rec); err != nil {
		return MeResponse{}, err
	}
	return s.Me(ctx, userID)
}

func (s *Service) SavePreferences(ctx context.Context, userID string, in PreferencesInput) (MeResponse, error) {
	prefs, err := validatePreferences(in)
	if err != nil {
		return MeResponse{}, err
	}
	if err := s.repo.UpsertPreferences(ctx, userID, prefs); err != nil {
		return MeResponse{}, err
	}
	return s.Me(ctx, userID)
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, lat, lng float64) error {
	cLat, cLng, err := validateLocation(lat, lng)
	if err != nil {
		return err
	}
	return s.repo.UpdateLocation(ctx, userID, cLat, cLng)
}

func (s *Service) SetDiscoverable(ctx context.Context, userID string, discoverable bool) error {
	return s.repo.SetDiscoverable(ctx, userID, discoverable)
}

// View returns one profile card if the viewer is allowed to see it.
func (s *Service) View(ctx context.Context, viewerID, targetID string) (Card, error) {
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return Card{}, err
	}
	if !ok {
		return Card{}, ErrForbidden
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return Card{}, err
	}
	if len(cards) == 0 {
		return Card{}, ErrForbidden
	}
	return cards[0], nil
}

// Cards loads public cards for userIDs in three queries (profiles, photos,
// interests), preserving the input order. Callers are responsible for
// authorization; this only shapes data.
func (s *Service) Cards(ctx context.Context, viewerID string, userIDs []string) ([]Card, error) {
	if len(userIDs) == 0 {
		return []Card{}, nil
	}
	viewer, err := s.repo.GetProfile(ctx, viewerID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	profiles, err := s.repo.LoadProfiles(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.LoadPhotos(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.LoadCardInterests(ctx, viewerID, userIDs)
	if err != nil {
		return nil, err
	}

	photosBy := map[string][]storedPhoto{}
	for _, p := range photos {
		photosBy[p.UserID] = append(photosBy[p.UserID], p)
	}
	interestsBy := map[string][]CardInterest{}
	for _, i := range interests {
		interestsBy[i.UserID] = append(interestsBy[i.UserID], i.CardInterest)
	}
	byID := map[string]storedProfile{}
	for _, p := range profiles {
		byID[p.UserID] = p
	}

	now := s.now()
	cards := make([]Card, 0, len(userIDs))
	for _, id := range userIDs {
		p, ok := byID[id]
		if !ok {
			continue
		}
		card := Card{
			UserID:     p.UserID,
			FirstName:  p.FirstName,
			Age:        AgeOn(p.Birthdate, now),
			Gender:     p.Gender,
			Bio:        p.Bio,
			JobTitle:   p.JobTitle,
			HeightCm:   p.HeightCm,
			LookingFor: p.LookingFor,
			City:       p.City,
			Interests:  nonNil(interestsBy[id]),
			Photos:     s.mapPhotos(photosBy[id]),
		}
		if viewer.Latitude != nil && p.Latitude != nil && viewer.UserID != p.UserID {
			d := ApproximateDistanceKm(HaversineKm(*viewer.Latitude, *viewer.Longitude, *p.Latitude, *p.Longitude))
			card.DistanceKm = &d
		}
		cards = append(cards, card)
	}
	return cards, nil
}

func (s *Service) mapPhotos(photos []storedPhoto) []Photo {
	out := make([]Photo, 0, len(photos))
	for _, p := range photos {
		out = append(out, Photo{ID: p.ID, URL: s.signer.URL(p.ID), Position: p.Position, Width: p.Width, Height: p.Height})
	}
	return out
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}
