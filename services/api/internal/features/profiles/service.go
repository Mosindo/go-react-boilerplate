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

// ValidationError carries a message that is safe to show to the end user.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalid(msg string) error { return &ValidationError{Message: msg} }

const (
	maxFirstNameRunes = 40
	maxBioRunes       = 500
	maxCityRunes      = 80
	maxInterests      = 10
	maxDistanceKm     = 500
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) Me(ctx context.Context, userID string) (MeResponse, error) {
	return s.repo.GetMe(ctx, userID)
}

func (s *Service) GetProfile(ctx context.Context, userID string) (MyProfile, error) {
	return s.repo.GetProfile(ctx, userID)
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (MyProfile, error) {
	in, err := s.validateProfile(req)
	if err != nil {
		return MyProfile{}, err
	}
	if err := s.repo.UpsertProfile(ctx, userID, in); err != nil {
		if errors.Is(err, ErrUnknownInterest) {
			return MyProfile{}, invalid("unknown interest")
		}
		return MyProfile{}, err
	}
	return s.repo.GetProfile(ctx, userID)
}

func (s *Service) validateProfile(req UpdateProfileRequest) (ProfileInput, error) {
	first, err := normalizeFirstName(req.FirstName)
	if err != nil {
		return ProfileInput{}, err
	}

	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return ProfileInput{}, invalid("birthDate must use the YYYY-MM-DD format")
	}
	age := AgeOn(birth, s.now().UTC())
	if age < MinAge {
		return ProfileInput{}, invalid("you must be at least 18 years old")
	}
	if age > MaxAge {
		return ProfileInput{}, invalid("birthDate is not plausible")
	}

	if !req.Gender.Valid() {
		return ProfileInput{}, invalid("gender must be one of man, woman, nonbinary")
	}

	bio := cleanText(req.Bio)
	if utf8.RuneCountInString(bio) > maxBioRunes {
		return ProfileInput{}, invalid("bio is limited to 500 characters")
	}
	city := cleanLine(req.City)
	if utf8.RuneCountInString(city) > maxCityRunes {
		return ProfileInput{}, invalid("city is limited to 80 characters")
	}

	if req.Latitude == nil || req.Longitude == nil {
		return ProfileInput{}, invalid("latitude and longitude are required")
	}
	lat, lon := *req.Latitude, *req.Longitude
	if math.IsNaN(lat) || math.IsNaN(lon) || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return ProfileInput{}, invalid("coordinates are out of range")
	}

	slugs := make([]string, 0, len(req.Interests))
	seen := make(map[string]struct{}, len(req.Interests))
	for _, slug := range req.Interests {
		slug = strings.ToLower(strings.TrimSpace(slug))
		if slug == "" || len(slug) > 40 {
			return ProfileInput{}, invalid("invalid interest")
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		slugs = append(slugs, slug)
	}
	if len(slugs) > maxInterests {
		return ProfileInput{}, invalid("you can pick at most 10 interests")
	}

	return ProfileInput{
		FirstName:     first,
		BirthDate:     birth,
		Gender:        req.Gender,
		Bio:           bio,
		City:          city,
		Latitude:      RoundCoordinate(lat),
		Longitude:     RoundCoordinate(lon),
		InterestSlugs: slugs,
		Discoverable:  req.Discoverable,
		ShowDistance:  req.ShowDistance,
	}, nil
}

func (s *Service) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	prefs, found, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return Preferences{}, err
	}
	if !found {
		return Preferences{InterestedIn: []Gender{}, MinAge: MinAge, MaxAge: 60, MaxDistanceKm: 50}, nil
	}
	return prefs, nil
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, p Preferences) (Preferences, error) {
	if len(p.InterestedIn) == 0 || len(p.InterestedIn) > len(allGenders) {
		return Preferences{}, invalid("pick at least one gender you are interested in")
	}
	seen := map[Gender]struct{}{}
	genders := make([]Gender, 0, len(p.InterestedIn))
	for _, g := range p.InterestedIn {
		if !g.Valid() {
			return Preferences{}, invalid("invalid gender preference")
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			genders = append(genders, g)
		}
	}
	p.InterestedIn = genders
	if p.MinAge < MinAge || p.MaxAge > MaxAge || p.MinAge > p.MaxAge {
		return Preferences{}, invalid("age range must be within 18-99 and minAge <= maxAge")
	}
	if p.MaxDistanceKm < 1 || p.MaxDistanceKm > maxDistanceKm {
		return Preferences{}, invalid("maxDistanceKm must be between 1 and 500")
	}
	if err := s.repo.UpsertPreferences(ctx, userID, p); err != nil {
		if errors.Is(err, ErrNotFound) {
			return Preferences{}, ErrNotFound
		}
		return Preferences{}, err
	}
	return p, nil
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) PublicProfile(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	return s.repo.GetPublicProfile(ctx, viewerID, targetID)
}

func normalizeFirstName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	if name == "" || utf8.RuneCountInString(name) > maxFirstNameRunes {
		return "", invalid("firstName must contain 1 to 40 characters")
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' && r != '’' && r != '.' {
			return "", invalid("firstName may only contain letters, spaces, hyphens and apostrophes")
		}
	}
	return name, nil
}

// cleanText trims and strips control characters while keeping line breaks.
func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, s)
	return strings.TrimSpace(s)
}

func cleanLine(s string) string {
	return strings.Join(strings.Fields(cleanText(s)), " ")
}
