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
	ErrNotFound        = errors.New("profile not found")
	ErrAlreadyExists   = errors.New("profile already exists")
	ErrUnderage        = errors.New("you must be at least 18 years old")
	ErrBirthDateChange = errors.New("birth date cannot be changed")
	ErrInvalidInterest = errors.New("unknown interest")
	ErrRequiresProfile = errors.New("create your profile first")
)

// ValidationError carries a user-facing message for a rejected field.
type ValidationError struct{ Field, Message string }

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

func invalid(field, msg string) error { return &ValidationError{Field: field, Message: msg} }

type Service struct {
	repo   Repository
	loader *Loader
	pool   Querier
	now    func() time.Time
}

func NewService(repo Repository, loader *Loader, pool Querier) *Service {
	return &Service{repo: repo, loader: loader, pool: pool, now: time.Now}
}

func (s *Service) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	stored, err := s.repo.GetOwn(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return OwnProfile{}, ErrNotFound
		}
		return OwnProfile{}, err
	}
	views, err := s.loader.Load(ctx, s.pool, userID, []string{userID})
	if err != nil {
		return OwnProfile{}, err
	}
	pub := views[userID]
	pub.DistanceKm = nil
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return OwnProfile{}, err
	}

	own := OwnProfile{
		PublicProfile: pub,
		BirthDate:     stored.BirthDate.Format("2006-01-02"),
		HasLocation:   stored.HasLocation,
		IsVisible:     stored.IsVisible,
		ShowDistance:  stored.ShowDistance,
		Preferences:   prefs,
		CreatedAt:     stored.CreatedAt,
		Missing:       []string{},
	}
	if prefs == nil {
		own.Missing = append(own.Missing, "preferences")
	}
	if !stored.HasLocation {
		own.Missing = append(own.Missing, "location")
	}
	if len(pub.Photos) == 0 {
		own.Missing = append(own.Missing, "photos")
	}
	own.IsComplete = len(own.Missing) == 0
	return own, nil
}

// Save creates the profile on first call, then updates it. The birth date is
// fixed at creation: later requests must send the same value.
func (s *Service) Save(ctx context.Context, userID string, req SaveProfileRequest) (OwnProfile, error) {
	in, err := s.validateProfile(req)
	if err != nil {
		return OwnProfile{}, err
	}
	birthDate, err := s.parseBirthDate(req.BirthDate)
	if err != nil {
		return OwnProfile{}, err
	}

	existing, err := s.repo.GetOwn(ctx, userID)
	switch {
	case errors.Is(err, ErrRepositoryNotFound):
		err = s.repo.Create(ctx, userID, in, birthDate)
	case err != nil:
		return OwnProfile{}, err
	default:
		if !existing.BirthDate.Equal(birthDate) {
			return OwnProfile{}, ErrBirthDateChange
		}
		err = s.repo.Update(ctx, userID, in)
	}
	if err != nil {
		if errors.Is(err, ErrRepositoryUnknownInterest) {
			return OwnProfile{}, ErrInvalidInterest
		}
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) SavePreferences(ctx context.Context, userID string, req SavePreferencesRequest) (OwnProfile, error) {
	seen := map[string]struct{}{}
	genders := make([]string, 0, 3)
	for _, g := range req.InterestedIn {
		g = strings.TrimSpace(strings.ToLower(g))
		if _, ok := validGenders[g]; !ok {
			return OwnProfile{}, invalid("interestedIn", "unknown gender")
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			genders = append(genders, g)
		}
	}
	if len(genders) == 0 {
		return OwnProfile{}, invalid("interestedIn", "choose at least one option")
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return OwnProfile{}, invalid("ageRange", "age range must be between 18 and 99")
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > 20000 {
		return OwnProfile{}, invalid("maxDistanceKm", "distance must be between 1 and 20000 km")
	}
	if _, err := s.repo.GetOwn(ctx, userID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return OwnProfile{}, ErrRequiresProfile
		}
		return OwnProfile{}, err
	}
	err := s.repo.SavePreferences(ctx, userID, Preferences{
		InterestedIn: genders, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm,
	})
	if err != nil {
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

// SaveLocation stores the position rounded to 2 decimals (~1 km). The exact
// coordinates sent by the device are never persisted.
func (s *Service) SaveLocation(ctx context.Context, userID string, req SaveLocationRequest) (OwnProfile, error) {
	if math.IsNaN(req.Latitude) || math.IsNaN(req.Longitude) ||
		req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		return OwnProfile{}, invalid("location", "invalid coordinates")
	}
	city, err := cleanText("city", req.City, 0, 80, false)
	if err != nil {
		return OwnProfile{}, err
	}
	lat := math.Round(req.Latitude*100) / 100
	lng := math.Round(req.Longitude*100) / 100
	if err := s.repo.SaveLocation(ctx, userID, lat, lng, city); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return OwnProfile{}, ErrRequiresProfile
		}
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) SavePrivacy(ctx context.Context, userID string, req SavePrivacyRequest) (OwnProfile, error) {
	if err := s.repo.SavePrivacy(ctx, userID, req.IsVisible, req.ShowDistance); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return OwnProfile{}, ErrRequiresProfile
		}
		return OwnProfile{}, err
	}
	return s.GetOwn(ctx, userID)
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) validateProfile(req SaveProfileRequest) (profileInput, error) {
	name, err := cleanText("firstName", req.FirstName, 1, 40, false)
	if err != nil {
		return profileInput{}, err
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' && r != '’' && r != '.' {
			return profileInput{}, invalid("firstName", "only letters, spaces, hyphens and apostrophes are allowed")
		}
	}
	gender := strings.TrimSpace(strings.ToLower(req.Gender))
	if _, ok := validGenders[gender]; !ok {
		return profileInput{}, invalid("gender", "choose man, woman or nonbinary")
	}
	bio, err := cleanText("bio", req.Bio, 0, 500, true)
	if err != nil {
		return profileInput{}, err
	}
	job, err := cleanText("jobTitle", req.JobTitle, 0, 80, false)
	if err != nil {
		return profileInput{}, err
	}
	if len(req.Interests) > MaxInterests {
		return profileInput{}, invalid("interests", "at most 10 interests")
	}
	seen := map[string]struct{}{}
	slugs := make([]string, 0, len(req.Interests))
	for _, slug := range req.Interests {
		slug = strings.TrimSpace(strings.ToLower(slug))
		if slug == "" || len(slug) > 40 {
			return profileInput{}, invalid("interests", "invalid interest")
		}
		if _, dup := seen[slug]; !dup {
			seen[slug] = struct{}{}
			slugs = append(slugs, slug)
		}
	}
	return profileInput{FirstName: name, Gender: gender, Bio: bio, JobTitle: job, Interests: slugs}, nil
}

func (s *Service) parseBirthDate(raw string) (time.Time, error) {
	d, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, invalid("birthDate", "use the format YYYY-MM-DD")
	}
	age := AgeOn(d, s.now())
	if age < MinAge {
		return time.Time{}, ErrUnderage
	}
	if age > MaxAge {
		return time.Time{}, invalid("birthDate", "please enter a valid birth date")
	}
	return d, nil
}

// AgeOn returns the completed years between birth and now.
func AgeOn(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

// cleanText trims, normalises whitespace, and rejects control characters.
func cleanText(field, raw string, minLen, maxLen int, multiline bool) (string, error) {
	v := strings.TrimSpace(raw)
	if !utf8.ValidString(v) {
		return "", invalid(field, "invalid text")
	}
	for _, r := range v {
		if unicode.IsControl(r) && !(multiline && (r == '\n')) {
			return "", invalid(field, "contains forbidden characters")
		}
	}
	if !multiline {
		v = strings.Join(strings.Fields(v), " ")
	}
	n := utf8.RuneCountInString(v)
	if n < minLen {
		return "", invalid(field, "is required")
	}
	if n > maxLen {
		return "", invalid(field, "is too long")
	}
	return v, nil
}
