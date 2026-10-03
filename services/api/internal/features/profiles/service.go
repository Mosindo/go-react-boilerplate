package profiles

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrValidation = errors.New("validation error")
)

// ValidationError carries a user-facing message and matches ErrValidation.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string        { return e.Message }
func (e *ValidationError) Is(target error) bool { return target == ErrValidation }

func invalid(msg string) error { return &ValidationError{Message: msg} }

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// ApproxDistance converts an exact distance into a deliberately coarse integer value so the
// real position cannot be triangulated: <=1km → 1, <=10km → ceil, beyond → nearest 5km.
func ApproxDistance(km *float64) *int {
	if km == nil || math.IsNaN(*km) {
		return nil
	}
	var out int
	switch d := *km; {
	case d <= 1:
		out = 1
	case d <= 10:
		out = int(math.Ceil(d))
	default:
		out = int(math.Round(d/5) * 5)
	}
	return &out
}

// RoundCoordinate keeps ~1 km precision (two decimals); anything finer is never stored.
func RoundCoordinate(v float64) float64 { return math.Round(v*100) / 100 }

func AgeOn(birth, now time.Time) int {
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	return age
}

func cleanText(s string, maxRunes int, field string, allowNewlines bool) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > maxRunes {
		return "", invalid(field + " is too long")
	}
	for _, r := range s {
		if unicode.IsControl(r) && !(allowNewlines && (r == '\n')) {
			return "", invalid(field + " contains invalid characters")
		}
	}
	return s, nil
}

func validCoordinates(lat, lng float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lng) && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (OwnProfile, error) {
	name, err := cleanText(req.FirstName, 40, "first name", false)
	if err != nil {
		return OwnProfile{}, err
	}
	if name == "" {
		return OwnProfile{}, invalid("first name is required")
	}
	birth, err := time.Parse("2006-01-02", req.BirthDate)
	if err != nil {
		return OwnProfile{}, invalid("birth date must be formatted YYYY-MM-DD")
	}
	age := AgeOn(birth, s.now())
	if age < MinAge {
		return OwnProfile{}, invalid("you must be at least 18 years old")
	}
	if age > MaxAge {
		return OwnProfile{}, invalid("birth date is not valid")
	}
	if _, ok := validGenders[req.Gender]; !ok {
		return OwnProfile{}, invalid("gender must be man, woman or nonbinary")
	}
	bio, err := cleanText(req.Bio, 500, "bio", true)
	if err != nil {
		return OwnProfile{}, err
	}
	city, err := cleanText(req.City, 80, "city", false)
	if err != nil {
		return OwnProfile{}, err
	}

	seen := make(map[string]struct{}, len(req.Interests))
	interests := make([]string, 0, len(req.Interests))
	for _, slug := range req.Interests {
		slug = strings.ToLower(strings.TrimSpace(slug))
		if _, dup := seen[slug]; dup || slug == "" {
			continue
		}
		seen[slug] = struct{}{}
		interests = append(interests, slug)
	}
	if len(interests) > MaxInterests {
		return OwnProfile{}, invalid("you can pick up to 10 interests")
	}

	in := UpsertInput{
		UserID: userID, FirstName: name, BirthDate: birth, Gender: req.Gender,
		Bio: bio, City: city, Interests: interests,
	}
	if (req.Latitude == nil) != (req.Longitude == nil) {
		return OwnProfile{}, invalid("latitude and longitude must be provided together")
	}
	if req.Latitude != nil {
		if !validCoordinates(*req.Latitude, *req.Longitude) {
			return OwnProfile{}, invalid("coordinates are out of range")
		}
		lat, lng := RoundCoordinate(*req.Latitude), RoundCoordinate(*req.Longitude)
		in.Latitude, in.Longitude = &lat, &lng
	}

	if err := s.repo.Upsert(ctx, in); err != nil {
		if errors.Is(err, ErrUnknownInterest) {
			return OwnProfile{}, invalid("unknown interest")
		}
		return OwnProfile{}, err
	}
	return s.repo.GetOwn(ctx, userID)
}

func (s *Service) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	return s.repo.GetOwn(ctx, userID)
}

func (s *Service) SetLocation(ctx context.Context, userID string, lat, lng float64) error {
	if !validCoordinates(lat, lng) {
		return invalid("coordinates are out of range")
	}
	return s.repo.SetLocation(ctx, userID, RoundCoordinate(lat), RoundCoordinate(lng))
}

func (s *Service) SetPreferences(ctx context.Context, userID string, req PreferencesRequest) (Preferences, error) {
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, invalid("age range must be within 18-99 and min must not exceed max")
	}
	if req.MaxDistanceKm != nil && (*req.MaxDistanceKm < 1 || *req.MaxDistanceKm > 500) {
		return Preferences{}, invalid("distance must be between 1 and 500 km")
	}
	seen := map[string]struct{}{}
	genders := make([]string, 0, 3)
	for _, g := range req.InterestedIn {
		if _, ok := validGenders[g]; !ok {
			return Preferences{}, invalid("interestedIn must only contain man, woman or nonbinary")
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			genders = append(genders, g)
		}
	}
	p := Preferences{InterestedIn: genders, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}
	if err := s.repo.SetPreferences(ctx, userID, p); err != nil {
		return Preferences{}, err
	}
	return p, nil
}

func (s *Service) SetPrivacy(ctx context.Context, userID string, visible, showDistance bool) error {
	return s.repo.SetPrivacy(ctx, userID, visible, showDistance)
}

// GetPublic returns another user's profile or ErrNotFound when it is hidden, blocked or absent.
// A caller may always load their own profile this way.
func (s *Service) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	m, err := s.repo.GetPublic(ctx, viewerID, []string{targetID}, viewerID != targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	p, ok := m[targetID]
	if !ok {
		return PublicProfile{}, ErrNotFound
	}
	return p, nil
}

// PublicProfiles batch-loads profiles in the given order, skipping unavailable ones.
func (s *Service) PublicProfiles(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error) {
	m, err := s.repo.GetPublic(ctx, viewerID, ids, false)
	if err != nil {
		return nil, err
	}
	out := make([]PublicProfile, 0, len(ids))
	for _, id := range ids {
		if p, ok := m[id]; ok {
			out = append(out, p)
		}
	}
	return out, nil
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// FirstName returns a user's first name for use in notification texts.
func (s *Service) FirstName(ctx context.Context, userID string) (string, error) {
	return s.repo.FirstName(ctx, userID)
}
