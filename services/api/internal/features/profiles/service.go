package profiles

import (
	"context"
	"errors"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalidInput    = errors.New("invalid profile input")
	ErrProfileNotFound = errors.New("profile not found")
	ErrNoProfile       = errors.New("create a profile first")
)

// ValidationError carries a user-facing message and wraps ErrInvalidInput.
type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }
func (e ValidationError) Unwrap() error { return ErrInvalidInput }

func invalid(msg string) error { return ValidationError{Message: msg} }

var namePattern = regexp.MustCompile(`^[\p{L}][\p{L}\p{M}' .-]*$`)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

// AgeAt returns the age in whole years of someone born on birth, as of now.
func AgeAt(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

// RoundCoordinate keeps two decimals (~1.1 km): precise enough for matching
// radii, too coarse to locate a person.
func RoundCoordinate(v float64) float64 { return math.Round(v*100) / 100 }

// DistanceBucket converts a distance to the coarse value shown to users: 5 km
// steps, rounded up, never less than 5.
func DistanceBucket(km float64) int {
	b := int(math.Ceil(km/5.0)) * 5
	if b < 5 {
		b = 5
	}
	return b
}

func (s *Service) ValidateProfile(req ProfileInput, rawBirth string) (ProfileInput, error) {
	in := req
	in.FirstName = strings.Join(strings.Fields(in.FirstName), " ")
	if n := utf8.RuneCountInString(in.FirstName); n < 1 || n > MaxNameRunes || !namePattern.MatchString(in.FirstName) {
		return in, invalid("first name must be 1 to 40 letters")
	}
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(rawBirth))
	if err != nil {
		return in, invalid("birth date must use the YYYY-MM-DD format")
	}
	age := AgeAt(birth, s.now())
	if age < MinAge {
		return in, invalid("you must be at least 18 years old")
	}
	if age > MaxAge {
		return in, invalid("birth date is not valid")
	}
	in.BirthDate = birth
	if !contains(Genders, in.Gender) {
		return in, invalid("gender must be woman, man or nonbinary")
	}
	in.Bio = cleanText(in.Bio, true)
	if utf8.RuneCountInString(in.Bio) > MaxBioRunes {
		return in, invalid("bio must be at most 500 characters")
	}
	in.City = cleanText(in.City, false)
	if utf8.RuneCountInString(in.City) > MaxCityRunes {
		return in, invalid("city must be at most 80 characters")
	}
	if math.IsNaN(in.Latitude) || math.IsNaN(in.Longitude) || in.Latitude < -90 || in.Latitude > 90 || in.Longitude < -180 || in.Longitude > 180 {
		return in, invalid("location is not valid")
	}
	in.Latitude, in.Longitude = RoundCoordinate(in.Latitude), RoundCoordinate(in.Longitude)
	slugs, err := normalizeSlugs(in.Interests)
	if err != nil {
		return in, err
	}
	in.Interests = slugs
	return in, nil
}

func normalizeSlugs(raw []string) ([]string, error) {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, slug := range raw {
		slug = strings.ToLower(strings.TrimSpace(slug))
		if slug == "" {
			continue
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		out = append(out, slug)
	}
	if len(out) > MaxInterests {
		return nil, invalid("choose at most 10 interests")
	}
	return out, nil
}

func (s *Service) ValidatePreferences(p Preferences) (Preferences, error) {
	seen := map[string]bool{}
	var interested []string
	for _, g := range p.InterestedIn {
		g = strings.ToLower(strings.TrimSpace(g))
		if !contains(Genders, g) {
			return p, invalid("interestedIn must contain woman, man or nonbinary")
		}
		if !seen[g] {
			seen[g] = true
			interested = append(interested, g)
		}
	}
	if len(interested) == 0 {
		return p, invalid("choose at least one gender")
	}
	sort.Strings(interested)
	if p.MinAge < MinAge || p.MaxAge > MaxAge || p.MinAge > p.MaxAge {
		return p, invalid("age range must be between 18 and 99 with min <= max")
	}
	if p.MaxDistanceKm < 1 || p.MaxDistanceKm > MaxDistanceKm {
		return p, invalid("distance must be between 1 and 500 km")
	}
	p.InterestedIn = interested
	return p, nil
}

func (s *Service) SaveProfile(ctx context.Context, userID string, in ProfileInput) error {
	return s.repo.UpsertProfile(ctx, userID, in)
}

func (s *Service) SavePreferences(ctx context.Context, userID string, p Preferences) error {
	return s.repo.UpsertPreferences(ctx, userID, p)
}

func (s *Service) SetPrivacy(ctx context.Context, userID string, discoverable, showDistance bool) error {
	return s.repo.SetPrivacy(ctx, userID, discoverable, showDistance)
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) Mine(ctx context.Context, userID string) (MyProfileResponse, error) {
	resp, err := s.repo.GetMine(ctx, userID)
	if err != nil {
		return resp, err
	}
	if resp.Profile != nil {
		if birth, perr := time.Parse("2006-01-02", resp.Profile.BirthDate); perr == nil {
			resp.Profile.Age = AgeAt(birth, s.now())
		}
	}
	resp.OnboardingComplete = resp.Profile != nil && len(resp.Photos) > 0
	return resp, nil
}

// View returns another user's profile if the viewer is allowed to see it:
// not blocked in either direction, and either discoverable, already matched,
// or the viewer themself.
func (s *Service) View(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	p, err := s.repo.GetPublic(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	return p, nil
}

func cleanText(s string, keepNewlines bool) string {
	s = strings.Map(func(r rune) rune {
		if r == '\n' && keepNewlines {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}
