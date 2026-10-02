package profiles

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/photos"
)

var (
	ErrInvalid         = errors.New("invalid profile")
	ErrUnderage        = errors.New("you must be at least 18 years old")
	ErrBirthDateLocked = errors.New("birth date cannot be changed")
	ErrNotFound        = errors.New("profile not found")
)

// ValidationError carries a user-facing message and the offending field.
type ValidationError struct{ Field, Message string }

func (e *ValidationError) Error() string   { return e.Field + ": " + e.Message }
func (e *ValidationError) Is(t error) bool { return t == ErrInvalid }

type PhotoSource interface {
	ListForUsers(ctx context.Context, userIDs []string) (map[string][]photos.View, error)
}

type Service struct {
	repo   Repository
	photos PhotoSource
	now    func() time.Time
}

func NewService(repo Repository, ph PhotoSource) *Service {
	return &Service{repo: repo, photos: ph, now: time.Now}
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// Status returns the owner's profile, preferences and what is still missing for onboarding.
func (s *Service) Status(ctx context.Context, userID string) (Status, error) {
	st := Status{Missing: []string{}}
	stored, err := s.repo.GetProfile(ctx, userID)
	switch {
	case errors.Is(err, ErrRepoNotFound):
		st.Missing = append(st.Missing, "profile", "photos")
		return st, nil
	case err != nil:
		return st, err
	}
	ph, err := s.photos.ListForUsers(ctx, []string{userID})
	if err != nil {
		return st, err
	}
	ints, err := s.repo.InterestsByUsers(ctx, []string{userID})
	if err != nil {
		return st, err
	}
	own := &Own{
		FirstName: stored.FirstName, BirthDate: stored.BirthDate.Format("2006-01-02"), Age: stored.Age,
		Gender: stored.Gender, Bio: stored.Bio, City: stored.City, Latitude: stored.Latitude, Longitude: stored.Longitude,
		Discoverable: stored.Discoverable, ShowDistance: stored.ShowDistance,
		Interests: nonNilInterests(ints[userID]), Photos: nonNilPhotos(ph[userID]),
	}
	st.Profile = own
	if prefs, err := s.repo.GetPreferences(ctx, userID); err == nil {
		st.Preferences = &prefs
	} else if !errors.Is(err, ErrRepoNotFound) {
		return st, err
	}
	if len(own.Photos) == 0 {
		st.Missing = append(st.Missing, "photos")
	}
	if st.Preferences == nil {
		st.Missing = append(st.Missing, "preferences")
	}
	st.Complete = len(st.Missing) == 0
	return st, nil
}

func (s *Service) Save(ctx context.Context, userID string, in ProfileInput) error {
	name := strings.TrimSpace(in.FirstName)
	if n := utf8.RuneCountInString(name); n < 1 || n > MaxFirstName || hasControl(name) {
		return &ValidationError{"firstName", "first name must be 1 to 50 characters"}
	}
	gender := strings.ToLower(strings.TrimSpace(in.Gender))
	if !contains(Genders, gender) {
		return &ValidationError{"gender", "unknown gender"}
	}
	bio := strings.TrimSpace(in.Bio)
	if utf8.RuneCountInString(bio) > MaxBioLen || hasBadControl(bio) {
		return &ValidationError{"bio", "bio must be at most 500 characters"}
	}
	city := strings.TrimSpace(in.City)
	if utf8.RuneCountInString(city) > MaxCityLen || hasControl(city) {
		return &ValidationError{"city", "city must be at most 80 characters"}
	}
	if math.IsNaN(in.Latitude) || math.IsNaN(in.Longitude) || in.Latitude < -90 || in.Latitude > 90 || in.Longitude < -180 || in.Longitude > 180 {
		return &ValidationError{"location", "invalid coordinates"}
	}
	if in.Latitude == 0 && in.Longitude == 0 {
		return &ValidationError{"location", "location is required"}
	}
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(in.BirthDate))
	if err != nil {
		return &ValidationError{"birthDate", "birth date must be YYYY-MM-DD"}
	}
	if AgeOn(birth, s.now()) < MinAge {
		return ErrUnderage
	}
	if AgeOn(birth, s.now()) > MaxAge {
		return &ValidationError{"birthDate", "invalid birth date"}
	}
	slugs, err := s.cleanInterests(ctx, in.Interests)
	if err != nil {
		return err
	}

	existing, err := s.repo.GetProfile(ctx, userID)
	switch {
	case err == nil:
		if !existing.BirthDate.Equal(birth) {
			return ErrBirthDateLocked // prevents age-gaming after sign-up
		}
	case !errors.Is(err, ErrRepoNotFound):
		return err
	}
	discoverable, showDistance := true, true
	if err == nil {
		discoverable, showDistance = existing.Discoverable, existing.ShowDistance
	}
	if in.Discoverable != nil {
		discoverable = *in.Discoverable
	}
	if in.ShowDistance != nil {
		showDistance = *in.ShowDistance
	}
	return s.repo.UpsertProfile(ctx, StoredProfile{
		UserID: userID, FirstName: name, BirthDate: birth, Gender: gender, Bio: bio, City: city,
		Latitude: roundCoord(in.Latitude), Longitude: roundCoord(in.Longitude),
		Discoverable: discoverable, ShowDistance: showDistance,
	}, slugs)
}

func (s *Service) SavePreferences(ctx context.Context, userID string, p Preferences) (Preferences, error) {
	seen := map[string]bool{}
	clean := make([]string, 0, 3)
	for _, g := range p.InterestedIn {
		g = strings.ToLower(strings.TrimSpace(g))
		if !contains(Genders, g) {
			return Preferences{}, &ValidationError{"interestedIn", "unknown gender"}
		}
		if !seen[g] {
			seen[g] = true
			clean = append(clean, g)
		}
	}
	if len(clean) == 0 {
		return Preferences{}, &ValidationError{"interestedIn", "choose at least one option"}
	}
	if p.MinAge < MinAge || p.MaxAge > MaxAge || p.MinAge > p.MaxAge {
		return Preferences{}, &ValidationError{"age", "age range must be between 18 and 99"}
	}
	if p.MaxDistanceKm != nil && (*p.MaxDistanceKm < 1 || *p.MaxDistanceKm > 20000) {
		return Preferences{}, &ValidationError{"maxDistanceKm", "distance must be between 1 and 20000 km"}
	}
	p.InterestedIn = clean
	if err := s.repo.UpsertPreferences(ctx, userID, p); err != nil {
		if errors.Is(err, ErrRepoNotFound) {
			return Preferences{}, ErrNotFound
		}
		return Preferences{}, err
	}
	return p, nil
}

// Cards builds public cards for the given users, in the given order, using three batched queries.
// Unknown ids are silently skipped. Distances are bucketed to 5 km so exact positions cannot be inferred.
func (s *Service) Cards(ctx context.Context, viewerID string, ids []string) ([]Card, error) {
	if len(ids) == 0 {
		return []Card{}, nil
	}
	rows, err := s.repo.CardRows(ctx, viewerID, ids)
	if err != nil {
		return nil, err
	}
	ph, err := s.photos.ListForUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	ints, err := s.repo.InterestsByUsers(ctx, append(append([]string{}, ids...), viewerID))
	if err != nil {
		return nil, err
	}
	mine := map[string]bool{}
	for _, i := range ints[viewerID] {
		mine[i.Slug] = true
	}
	byID := make(map[string]Card, len(rows))
	for _, r := range rows {
		card := Card{ID: r.UserID, FirstName: r.FirstName, Age: r.Age, Gender: r.Gender, Bio: r.Bio, City: r.City,
			Photos: nonNilPhotos(ph[r.UserID])}
		if r.DistanceKm != nil {
			d := ApproxDistance(*r.DistanceKm)
			card.DistanceKm = &d
		}
		card.Interests = []Interest{}
		for _, i := range ints[r.UserID] {
			i.Shared = mine[i.Slug]
			card.Interests = append(card.Interests, i)
		}
		byID[r.UserID] = card
	}
	out := make([]Card, 0, len(ids))
	for _, id := range ids {
		if c, ok := byID[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *Service) cleanInterests(ctx context.Context, in []string) ([]string, error) {
	seen := map[string]bool{}
	slugs := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.ToLower(strings.TrimSpace(v))
		if v != "" && !seen[v] {
			seen[v] = true
			slugs = append(slugs, v)
		}
	}
	if len(slugs) > MaxInterests {
		return nil, &ValidationError{"interests", "choose at most 10 interests"}
	}
	if len(slugs) > 0 {
		n, err := s.repo.CountKnownInterests(ctx, slugs)
		if err != nil {
			return nil, err
		}
		if n != len(slugs) {
			return nil, &ValidationError{"interests", "unknown interest"}
		}
	}
	return slugs, nil
}

// AgeOn returns the completed years between birth and now.
func AgeOn(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

// ApproxDistance rounds up to the next 5 km (minimum 5).
func ApproxDistance(km float64) int {
	d := int(math.Ceil(km/5)) * 5
	if d < 5 {
		d = 5
	}
	return d
}

func roundCoord(v float64) float64 { return math.Round(v*coordPrecision) / coordPrecision }

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func hasControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// hasBadControl allows newlines and tabs in free text.
func hasBadControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return true
		}
	}
	return false
}

func nonNilPhotos(p []photos.View) []photos.View {
	if p == nil {
		return []photos.View{}
	}
	return p
}

func nonNilInterests(i []Interest) []Interest {
	if i == nil {
		return []Interest{}
	}
	return i
}
