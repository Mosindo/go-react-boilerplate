package profiles

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/api/internal/platform/mediaurl"
)

var (
	ErrProfileNotFound   = errors.New("profile not found")
	ErrInvalidFirstName  = errors.New("first name must contain 1 to 40 characters")
	ErrInvalidBirthdate  = errors.New("birthdate must be a valid date formatted YYYY-MM-DD")
	ErrUnderage          = errors.New("you must be at least 18 years old")
	ErrBirthdateLocked   = errors.New("birthdate cannot be changed once the profile is created")
	ErrInvalidGender     = errors.New("invalid gender")
	ErrInvalidText       = errors.New("text field too long")
	ErrInvalidHeight     = errors.New("height must be between 120 and 230 cm")
	ErrInvalidGoal       = errors.New("invalid relationship goal")
	ErrInvalidInterests  = errors.New("invalid interests selection")
	ErrInvalidPrefs      = errors.New("invalid preferences")
	ErrInvalidLocation   = errors.New("invalid coordinates")
	ErrProfileNotVisible = errors.New("profile not visible")
)

type Service struct {
	repo   Repository
	signer *mediaurl.Signer
	now    func() time.Time
}

func NewService(repo Repository, signer *mediaurl.Signer) *Service {
	return &Service{repo: repo, signer: signer, now: time.Now}
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) GetOwn(ctx context.Context, userID string) (Profile, error) {
	stored, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Profile{}, ErrProfileNotFound
		}
		return Profile{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return Profile{}, err
	}
	interests, err := s.repo.InterestsByUserIDs(ctx, []string{userID})
	if err != nil {
		return Profile{}, err
	}
	photos, err := s.repo.PhotosByUserIDs(ctx, []string{userID})
	if err != nil {
		return Profile{}, err
	}
	profile := Profile{
		UserID:           stored.UserID,
		FirstName:        stored.FirstName,
		Birthdate:        stored.Birthdate,
		Gender:           stored.Gender,
		Bio:              stored.Bio,
		JobTitle:         stored.JobTitle,
		City:             stored.City,
		HeightCm:         stored.HeightCm,
		RelationshipGoal: stored.RelationshipGoal,
		Latitude:         stored.Latitude,
		Longitude:        stored.Longitude,
		Discoverable:     stored.Discoverable,
		Interests:        interests[userID],
		Photos:           s.photoRefs(photos[userID]),
		Preferences:      prefs,
	}
	if profile.Interests == nil {
		profile.Interests = []Interest{}
	}
	return profile, nil
}

// ValidateProfileInput normalizes and validates profile fields. It is pure so
// it can be unit tested without a database.
func ValidateProfileInput(in UpsertProfileInput, now time.Time) (UpsertProfileInput, time.Time, error) {
	out := in
	out.FirstName = strings.Join(strings.Fields(in.FirstName), " ")
	if n := utf8.RuneCountInString(out.FirstName); n < 1 || n > maxFirstName {
		return out, time.Time{}, ErrInvalidFirstName
	}
	birthdate, err := time.Parse(birthdateLayout, strings.TrimSpace(in.Birthdate))
	if err != nil {
		return out, time.Time{}, ErrInvalidBirthdate
	}
	age := AgeOn(birthdate, now)
	if age < MinAge {
		return out, time.Time{}, ErrUnderage
	}
	if age > 120 {
		return out, time.Time{}, ErrInvalidBirthdate
	}
	if !slices.Contains(Genders, in.Gender) {
		return out, time.Time{}, ErrInvalidGender
	}
	out.Bio = strings.TrimSpace(in.Bio)
	out.JobTitle = strings.TrimSpace(in.JobTitle)
	out.City = strings.TrimSpace(in.City)
	if utf8.RuneCountInString(out.Bio) > maxBio || utf8.RuneCountInString(out.JobTitle) > maxJobTitle || utf8.RuneCountInString(out.City) > maxCity {
		return out, time.Time{}, ErrInvalidText
	}
	if in.HeightCm != nil && (*in.HeightCm < minHeightCm || *in.HeightCm > maxHeightCm) {
		return out, time.Time{}, ErrInvalidHeight
	}
	if in.RelationshipGoal != nil {
		goal := strings.TrimSpace(*in.RelationshipGoal)
		if goal == "" {
			out.RelationshipGoal = nil
		} else if !slices.Contains(RelationshipGoals, goal) {
			return out, time.Time{}, ErrInvalidGoal
		} else {
			out.RelationshipGoal = &goal
		}
	}
	unique := make([]int, 0, len(in.InterestIDs))
	for _, id := range in.InterestIDs {
		if id <= 0 || id > 32767 {
			return out, time.Time{}, ErrInvalidInterests
		}
		if !slices.Contains(unique, id) {
			unique = append(unique, id)
		}
	}
	if len(unique) > MaxInterests {
		return out, time.Time{}, ErrInvalidInterests
	}
	out.InterestIDs = unique
	return out, birthdate, nil
}

func (s *Service) Upsert(ctx context.Context, userID string, in UpsertProfileInput) (Profile, error) {
	normalized, birthdate, err := ValidateProfileInput(in, s.now())
	if err != nil {
		return Profile{}, err
	}
	existing, err := s.repo.GetProfile(ctx, userID)
	switch {
	case err == nil:
		if !existing.Birthdate.Equal(birthdate) {
			return Profile{}, ErrBirthdateLocked
		}
	case errors.Is(err, ErrRepositoryNotFound):
	default:
		return Profile{}, err
	}
	if err := s.repo.UpsertProfile(ctx, userID, normalized, birthdate); err != nil {
		if errors.Is(err, ErrRepositoryUnknownInterest) {
			return Profile{}, ErrInvalidInterests
		}
		return Profile{}, err
	}
	return s.GetOwn(ctx, userID)
}

// ValidatePreferences checks the dating preferences.
func ValidatePreferences(prefs Preferences) (Preferences, error) {
	out := prefs
	seen := make([]string, 0, len(prefs.InterestedIn))
	for _, g := range prefs.InterestedIn {
		if !slices.Contains(Genders, g) {
			return out, ErrInvalidPrefs
		}
		if !slices.Contains(seen, g) {
			seen = append(seen, g)
		}
	}
	if len(seen) == 0 {
		return out, ErrInvalidPrefs
	}
	out.InterestedIn = seen
	if prefs.AgeMin < MinAge || prefs.AgeMax > MaxAge || prefs.AgeMin > prefs.AgeMax {
		return out, ErrInvalidPrefs
	}
	if prefs.MaxDistanceKm < MinDistanceKm || prefs.MaxDistanceKm > MaxDistanceKm {
		return out, ErrInvalidPrefs
	}
	return out, nil
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, prefs Preferences) (Preferences, error) {
	normalized, err := ValidatePreferences(prefs)
	if err != nil {
		return Preferences{}, err
	}
	if _, err := s.repo.GetProfile(ctx, userID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return Preferences{}, ErrProfileNotFound
		}
		return Preferences{}, err
	}
	if err := s.repo.UpsertPreferences(ctx, userID, normalized); err != nil {
		return Preferences{}, err
	}
	return normalized, nil
}

// UpdateLocation stores a coarse location only (see SnapCoordinate).
func (s *Service) UpdateLocation(ctx context.Context, userID string, lat, lng float64) error {
	if !ValidCoordinates(lat, lng) {
		return ErrInvalidLocation
	}
	snappedLat, snappedLng := SnapCoordinate(lat), SnapCoordinate(lng)
	if err := s.repo.SetLocation(ctx, userID, &snappedLat, &snappedLng); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrProfileNotFound
		}
		return err
	}
	return nil
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	if err := s.repo.SetLocation(ctx, userID, nil, nil); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrProfileNotFound
		}
		return err
	}
	return nil
}

func (s *Service) SetDiscoverable(ctx context.Context, userID string, discoverable bool) error {
	if err := s.repo.SetDiscoverable(ctx, userID, discoverable); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrProfileNotFound
		}
		return err
	}
	return nil
}

// ViewProfile returns another member's card if the viewer is allowed to see it.
func (s *Service) ViewProfile(ctx context.Context, viewerID, targetID string) (Card, error) {
	ok, err := s.repo.CanView(ctx, viewerID, targetID)
	if err != nil {
		return Card{}, err
	}
	if !ok {
		return Card{}, ErrProfileNotVisible
	}
	cards, err := s.Cards(ctx, viewerID, []string{targetID})
	if err != nil {
		return Card{}, err
	}
	card, found := cards[targetID]
	if !found {
		return Card{}, ErrProfileNotVisible
	}
	return card, nil
}

// Cards hydrates public cards for several users with a constant number of
// queries (no N+1). Visibility must be checked by the caller.
func (s *Service) Cards(ctx context.Context, viewerID string, userIDs []string) (map[string]Card, error) {
	cards := make(map[string]Card, len(userIDs))
	if len(userIDs) == 0 {
		return cards, nil
	}
	ids := append([]string{viewerID}, userIDs...)
	stored, err := s.repo.ProfilesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	interests, err := s.repo.InterestsByUserIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	photos, err := s.repo.PhotosByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	var viewer *StoredProfile
	for i := range stored {
		if stored[i].UserID == viewerID {
			viewer = &stored[i]
		}
	}
	viewerInterests := map[int]bool{}
	for _, i := range interests[viewerID] {
		viewerInterests[i.ID] = true
	}

	now := s.now()
	for _, p := range stored {
		if p.UserID == viewerID && !slices.Contains(userIDs, viewerID) {
			continue
		}
		card := Card{
			UserID:           p.UserID,
			FirstName:        p.FirstName,
			Age:              AgeOn(p.Birthdate, now),
			Gender:           p.Gender,
			Bio:              p.Bio,
			JobTitle:         p.JobTitle,
			City:             p.City,
			HeightCm:         p.HeightCm,
			RelationshipGoal: p.RelationshipGoal,
			Photos:           s.photoRefs(photos[p.UserID]),
			IsDemo:           p.IsDemo,
		}
		if viewer != nil && viewer.Latitude != nil && p.Latitude != nil && p.UserID != viewerID {
			km := ApproximateKm(HaversineKm(*viewer.Latitude, *viewer.Longitude, *p.Latitude, *p.Longitude))
			card.DistanceKm = &km
		}
		for _, i := range interests[p.UserID] {
			card.Interests = append(card.Interests, CardInterest{ID: i.ID, Label: i.Label, Shared: p.UserID != viewerID && viewerInterests[i.ID]})
		}
		// Shared interests first: they are the best conversation starters.
		slices.SortStableFunc(card.Interests, func(a, b CardInterest) int {
			switch {
			case a.Shared && !b.Shared:
				return -1
			case !a.Shared && b.Shared:
				return 1
			default:
				return 0
			}
		})
		cards[p.UserID] = card
	}
	return cards, nil
}

func (s *Service) photoRefs(photos []StoredPhoto) []PhotoRef {
	refs := make([]PhotoRef, 0, len(photos))
	for _, p := range photos {
		refs = append(refs, PhotoRef{ID: p.ID, URL: s.signer.PhotoURL(p.ID), Position: p.Position})
	}
	return refs
}
