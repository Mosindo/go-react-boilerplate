package profiles

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
)

// ValidationError is a user-correctable input problem.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalid(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}

var ErrNotFound = errors.New("profile not found")

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) Overview(ctx context.Context, userID string) (Overview, error) {
	o, err := s.repo.Overview(ctx, userID)
	if err != nil {
		return Overview{}, err
	}
	if o.Profile != nil {
		o.Profile.Age = AgeOn(o.Profile.BirthDate, s.now())
	}
	o.Missing = []string{}
	if o.Profile == nil {
		o.Missing = append(o.Missing, "profile")
	}
	if o.Preferences == nil {
		o.Missing = append(o.Missing, "preferences")
	}
	if o.Profile == nil || len(o.Profile.Photos) == 0 {
		o.Missing = append(o.Missing, "photos")
	}
	o.OnboardingComplete = len(o.Missing) == 0
	return o, nil
}

func (s *Service) SaveProfile(ctx context.Context, userID string, in ProfileInput) error {
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.Bio = strings.TrimSpace(in.Bio)
	in.City = strings.TrimSpace(in.City)
	if n := len([]rune(in.FirstName)); n < 1 || n > 40 {
		return invalid("firstName must be 1-40 characters")
	}
	if hasControl(in.FirstName) || hasControl(in.City) {
		return invalid("invalid characters")
	}
	if !validGenders[in.Gender] {
		return invalid("invalid gender")
	}
	if len([]rune(in.Bio)) > 500 {
		return invalid("bio must be at most 500 characters")
	}
	if len([]rune(in.City)) > 80 {
		return invalid("city must be at most 80 characters")
	}
	birth, err := time.Parse("2006-01-02", in.BirthDate)
	if err != nil {
		return invalid("birthDate must be YYYY-MM-DD")
	}
	if age := AgeOn(birth.Format("2006-01-02"), s.now()); age < MinAge {
		return invalid("you must be at least %d years old", MinAge)
	} else if age > MaxAge {
		return invalid("invalid birthDate")
	}
	slugs, err := dedupe(in.Interests, maxInterests, "interests")
	if err != nil {
		return err
	}
	in.Interests = slugs
	if err := s.repo.UpsertProfile(ctx, userID, in, birth); err != nil {
		if errors.Is(err, ErrUnknownInterest) {
			return invalid("unknown interest")
		}
		return err
	}
	return nil
}

func (s *Service) SavePreferences(ctx context.Context, userID string, p Preferences) error {
	genders, err := dedupe(p.InterestedIn, 3, "interestedIn")
	if err != nil {
		return err
	}
	if len(genders) == 0 {
		return invalid("interestedIn is required")
	}
	for _, g := range genders {
		if !validGenders[g] {
			return invalid("invalid gender in interestedIn")
		}
	}
	p.InterestedIn = genders
	if p.MinAge < MinAge || p.MaxAge > MaxAge || p.MinAge > p.MaxAge {
		return invalid("age range must be within %d-%d and min <= max", MinAge, MaxAge)
	}
	if p.MaxDistanceKm != nil && (*p.MaxDistanceKm < 1 || *p.MaxDistanceKm > 20000) {
		return invalid("maxDistanceKm must be between 1 and 20000")
	}
	return s.repo.UpsertPreferences(ctx, userID, p)
}

func (s *Service) SavePrivacy(ctx context.Context, userID string, p Privacy) error {
	if err := s.repo.UpdatePrivacy(ctx, userID, p); err != nil {
		return err
	}
	return nil
}

// SaveLocation stores only a coarse position (2 decimals ≈ 1 km).
func (s *Service) SaveLocation(ctx context.Context, userID string, l LocationInput) error {
	if math.IsNaN(l.Latitude) || math.IsNaN(l.Longitude) ||
		l.Latitude < -90 || l.Latitude > 90 || l.Longitude < -180 || l.Longitude > 180 {
		return invalid("invalid coordinates")
	}
	return s.repo.UpdateLocation(ctx, userID, Coarse(l.Latitude), Coarse(l.Longitude))
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	return s.repo.ClearLocation(ctx, userID)
}

// Coarse rounds a coordinate to 2 decimals.
func Coarse(v float64) float64 { return math.Round(v*100) / 100 }

// AgeOn returns the age in full years on the given day for a YYYY-MM-DD birth date.
func AgeOn(birthDate string, now time.Time) int {
	b, err := time.Parse("2006-01-02", birthDate[:min(len(birthDate), 10)])
	if err != nil {
		return 0
	}
	age := now.Year() - b.Year()
	if now.Month() < b.Month() || (now.Month() == b.Month() && now.Day() < b.Day()) {
		age--
	}
	return age
}

func dedupe(values []string, max int, field string) ([]string, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	if len(out) > max {
		return nil, invalid("%s: at most %d values", field, max)
	}
	return out, nil
}

func hasControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
