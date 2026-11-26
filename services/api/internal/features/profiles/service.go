package profiles

import (
	"context"
	"errors"
	"sort"
	"strings"

	platformerrors "example.com/api/internal/platform/errors"
)

var (
	ErrProfileRequired = platformerrors.Conflict("profile_required", "Complétez d'abord votre profil.")
	ErrProfileNotFound = platformerrors.NotFound("profile_not_found", "Profil introuvable.")
	ErrBirthdateLocked = platformerrors.Conflict("birthdate_locked", "La date de naissance ne peut pas être modifiée. Contactez le support en cas d'erreur.")
	ErrUnknownInterest = platformerrors.Validation("invalid_interests", "Un ou plusieurs centres d'intérêt sont inconnus.")
	ErrTooManyInterest = platformerrors.Validation("too_many_interests", "Vous pouvez choisir jusqu'à 10 centres d'intérêt.")
	ErrInvalidGender   = platformerrors.Validation("invalid_gender", "Genre invalide.")
	ErrInvalidLocation = platformerrors.Validation("invalid_location", "Coordonnées invalides.")
)

// URLSigner turns a storage key into a signed, short-lived media URL.
type URLSigner interface {
	URL(key string) string
}

type Service struct {
	repo   Repository
	signer URLSigner
}

func NewService(repo Repository, signer URLSigner) *Service {
	return &Service{repo: repo, signer: signer}
}

func (s *Service) GetMine(ctx context.Context, userID string) (MyProfile, error) {
	profile, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return MyProfile{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return MyProfile{}, err
	}
	photos, err := s.repo.LoadPhotos(ctx, []string{userID})
	if err != nil {
		return MyProfile{}, err
	}
	interests, err := s.repo.LoadInterests(ctx, []string{userID})
	if err != nil {
		return MyProfile{}, err
	}

	out := MyProfile{
		UserID:       userID,
		Discoverable: true,
		ShowDistance: true,
		Interests:    nonNilInterests(interests[userID]),
		Photos:       s.photoViews(photos[userID]),
		Preferences:  prefs,
	}
	if profile != nil {
		out.FirstName = profile.FirstName
		out.Birthdate = profile.Birthdate.Format(birthdateLayout)
		out.Age = AgeOn(profile.Birthdate, today())
		out.Gender = profile.Gender
		out.Bio = profile.Bio
		out.JobTitle = profile.JobTitle
		out.City = profile.City
		out.HasLocation = profile.Latitude != nil
		out.Discoverable = profile.Discoverable
		out.ShowDistance = profile.ShowDistance
	}
	out.Completion = Completion{
		Profile:     profile != nil,
		Preferences: prefs != nil,
		Photos:      len(out.Photos) > 0,
	}
	out.Completion.Complete = out.Completion.Profile && out.Completion.Preferences && out.Completion.Photos
	return out, nil
}

func (s *Service) UpdateMine(ctx context.Context, userID string, req UpdateProfileRequest) (MyProfile, error) {
	firstName, err := NormalizeFirstName(req.FirstName)
	if err != nil {
		return MyProfile{}, err
	}
	birthdate, err := ParseBirthdate(req.Birthdate)
	if err != nil {
		return MyProfile{}, err
	}
	gender := strings.TrimSpace(req.Gender)
	if !ValidGender(gender) {
		return MyProfile{}, ErrInvalidGender
	}
	bio, err := NormalizeText(req.Bio, maxBioLen, "invalid_bio", "La bio ne peut pas dépasser 500 caractères.")
	if err != nil {
		return MyProfile{}, err
	}
	job, err := NormalizeText(strings.ReplaceAll(req.JobTitle, "\n", " "), maxJobTitleLen, "invalid_job_title", "Le métier ne peut pas dépasser 60 caractères.")
	if err != nil {
		return MyProfile{}, err
	}
	city, err := NormalizeText(strings.ReplaceAll(req.City, "\n", " "), maxCityLen, "invalid_city", "La ville ne peut pas dépasser 60 caractères.")
	if err != nil {
		return MyProfile{}, err
	}
	slugs := dedupe(req.Interests)
	if len(slugs) > MaxInterests {
		return MyProfile{}, ErrTooManyInterest
	}

	existing, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return MyProfile{}, err
	}
	if existing != nil && !existing.Birthdate.Equal(birthdate) {
		return MyProfile{}, ErrBirthdateLocked
	}

	err = s.repo.UpsertProfile(ctx, storedProfile{
		UserID: userID, FirstName: firstName, Birthdate: birthdate, Gender: gender,
		Bio: bio, JobTitle: job, City: city,
	}, slugs)
	if errors.Is(err, errRepoUnknownInterest) {
		return MyProfile{}, ErrUnknownInterest
	}
	if err != nil {
		return MyProfile{}, err
	}
	return s.GetMine(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (MyProfile, error) {
	prefs, err := ValidatePreferences(req)
	if err != nil {
		return MyProfile{}, err
	}
	if err := s.repo.UpsertPreferences(ctx, userID, prefs); err != nil {
		return MyProfile{}, err
	}
	return s.GetMine(ctx, userID)
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, lat, lng float64) error {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 || (lat == 0 && lng == 0) {
		return ErrInvalidLocation
	}
	cLat, cLng := CoarsenCoordinate(lat), CoarsenCoordinate(lng)
	return s.mapNotFound(s.repo.UpdateLocation(ctx, userID, &cLat, &cLng))
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	return s.mapNotFound(s.repo.UpdateLocation(ctx, userID, nil, nil))
}

func (s *Service) UpdatePrivacy(ctx context.Context, userID string, req UpdatePrivacyRequest) (MyProfile, error) {
	if err := s.mapNotFound(s.repo.UpdatePrivacy(ctx, userID, req.Discoverable, req.ShowDistance)); err != nil {
		return MyProfile{}, err
	}
	return s.GetMine(ctx, userID)
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// GetPublic returns another member's profile if the visibility rule allows it.
func (s *Service) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if !ok {
		return PublicProfile{}, ErrProfileNotFound
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	card, found := cards[targetID]
	if !found {
		return PublicProfile{}, ErrProfileNotFound
	}
	return card, nil
}

// Cards builds public profiles for a batch of users in a constant number of queries.
// Callers are responsible for having authorised the viewer to see these users.
func (s *Service) Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]PublicProfile, error) {
	out := make(map[string]PublicProfile, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	all := append([]string{viewerID}, userIDs...)
	profiles, err := s.repo.LoadProfiles(ctx, all)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.LoadPhotos(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.LoadInterests(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	viewer, hasViewer := profiles[viewerID]
	now := today()
	for _, id := range userIDs {
		p, ok := profiles[id]
		if !ok {
			continue
		}
		card := PublicProfile{
			UserID:    p.UserID,
			FirstName: p.FirstName,
			Age:       AgeOn(p.Birthdate, now),
			Gender:    p.Gender,
			Bio:       p.Bio,
			JobTitle:  p.JobTitle,
			City:      p.City,
			Interests: nonNilInterests(interests[id]),
			Photos:    s.photoViews(photos[id]),
			IsDemo:    p.IsDemo,
		}
		if id != viewerID && p.ShowDistance && hasViewer && viewer.Latitude != nil && p.Latitude != nil {
			d := ApproxDistanceKm(HaversineKm(*viewer.Latitude, *viewer.Longitude, *p.Latitude, *p.Longitude))
			card.DistanceKm = &d
		}
		out[id] = card
	}
	return out, nil
}

func (s *Service) photoViews(photos []storedPhoto) []PhotoView {
	views := make([]PhotoView, 0, len(photos))
	for _, p := range photos {
		views = append(views, PhotoView{ID: p.ID, URL: s.signer.URL(p.StorageKey), Width: p.Width, Height: p.Height, Position: p.Position})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].Position < views[j].Position })
	return views
}

func (s *Service) mapNotFound(err error) error {
	if errors.Is(err, errRepoNotFound) {
		return ErrProfileRequired
	}
	return err
}

func nonNilInterests(in []Interest) []Interest {
	if in == nil {
		return []Interest{}
	}
	return in
}

func dedupe(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}
