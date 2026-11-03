package profiles

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
)

const (
	MinAge        = 18
	MaxAge        = 120
	maxInterests  = 10
	maxFirstName  = 40
	maxBio        = 500
	maxCity       = 80
	MaxDistanceKm = 500 // values at or above this mean "anywhere"
)

var validGenders = map[string]bool{"man": true, "woman": true, "nonbinary": true}

type Service struct {
	repo *Repository
	now  func() time.Time
}

func NewService(repo *Repository) *Service { return &Service{repo: repo, now: time.Now} }

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// AgeOn computes the age in whole years on the given day.
func AgeOn(birth, today time.Time) int {
	age := today.Year() - birth.Year()
	if today.Month() < birth.Month() || (today.Month() == birth.Month() && today.Day() < birth.Day()) {
		age--
	}
	return age
}

func cleanText(raw string, max int, field string, required bool) (string, error) {
	v := strings.TrimSpace(raw)
	if required && v == "" {
		return "", platformerrors.Invalid(field + " is required")
	}
	if utf8.RuneCountInString(v) > max {
		return "", platformerrors.Invalid(field + " is too long")
	}
	for _, r := range v {
		if unicode.IsControl(r) && r != '\n' {
			return "", platformerrors.Invalid(field + " contains invalid characters")
		}
	}
	return v, nil
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (MyProfile, error) {
	first, err := cleanText(req.FirstName, maxFirstName, "firstName", true)
	if err != nil {
		return MyProfile{}, err
	}
	bio, err := cleanText(req.Bio, maxBio, "bio", false)
	if err != nil {
		return MyProfile{}, err
	}
	city, err := cleanText(req.City, maxCity, "city", false)
	if err != nil {
		return MyProfile{}, err
	}
	gender := strings.ToLower(strings.TrimSpace(req.Gender))
	if !validGenders[gender] {
		return MyProfile{}, platformerrors.Invalid("gender must be man, woman or nonbinary")
	}
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return MyProfile{}, platformerrors.Invalid("birthDate must be YYYY-MM-DD")
	}
	age := AgeOn(birth, s.now().UTC())
	if age < MinAge {
		return MyProfile{}, platformerrors.New(403, "underage", "you must be at least 18 years old")
	}
	if age > MaxAge {
		return MyProfile{}, platformerrors.Invalid("birthDate is not valid")
	}
	interests := dedupe(req.Interests)
	if len(interests) > maxInterests {
		return MyProfile{}, platformerrors.Invalid("too many interests")
	}
	if len(interests) > 0 {
		n, err := s.repo.CountKnownInterests(ctx, interests)
		if err != nil {
			return MyProfile{}, err
		}
		if n != len(interests) {
			return MyProfile{}, platformerrors.Invalid("unknown interest")
		}
	}
	req.FirstName, req.Bio, req.City, req.Gender = first, bio, city, gender
	if err := s.repo.Upsert(ctx, userID, req, birth, interests); err != nil {
		return MyProfile{}, err
	}
	return s.Mine(ctx, userID)
}

func (s *Service) Mine(ctx context.Context, userID string) (MyProfile, error) {
	me, err := s.repo.GetMine(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return MyProfile{}, platformerrors.New(404, "profile_not_found", "profile not found")
	}
	if err != nil {
		return MyProfile{}, err
	}
	me.Missing = []string{}
	if len(me.Photos) == 0 {
		me.Missing = append(me.Missing, "photos")
	}
	if me.Preferences == nil {
		me.Missing = append(me.Missing, "preferences")
	}
	me.Complete = len(me.Missing) == 0
	return me, nil
}

// SetLocation stores a coarse position (rounded to ~1 km). Exact GPS is
// never persisted.
func (s *Service) SetLocation(ctx context.Context, userID string, req LocationRequest) error {
	if math.IsNaN(req.Latitude) || math.IsNaN(req.Longitude) ||
		req.Latitude < -90 || req.Latitude > 90 || req.Longitude < -180 || req.Longitude > 180 {
		return platformerrors.Invalid("invalid coordinates")
	}
	err := s.repo.SetLocation(ctx, userID, round2(req.Latitude), round2(req.Longitude))
	if errors.Is(err, ErrNotFound) {
		return platformerrors.New(404, "profile_not_found", "create your profile first")
	}
	return err
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	return s.repo.ClearLocation(ctx, userID)
}

func (s *Service) SetPrivacy(ctx context.Context, userID string, req PrivacyRequest) error {
	err := s.repo.SetPrivacy(ctx, userID, req.ShowDistance, req.Discoverable)
	if errors.Is(err, ErrNotFound) {
		return platformerrors.New(404, "profile_not_found", "create your profile first")
	}
	return err
}

func (s *Service) SetPreferences(ctx context.Context, userID string, req PreferencesRequest) error {
	genders := dedupe(req.InterestedIn)
	if len(genders) == 0 {
		return platformerrors.Invalid("interestedIn is required")
	}
	for _, g := range genders {
		if !validGenders[g] {
			return platformerrors.Invalid("interestedIn contains an invalid value")
		}
	}
	if req.AgeMin < MinAge || req.AgeMax > MaxAge || req.AgeMin > req.AgeMax {
		return platformerrors.Invalid("invalid age range")
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > MaxDistanceKm {
		return platformerrors.Invalid("invalid maxDistanceKm")
	}
	return s.repo.SetPreferences(ctx, userID, Preferences{
		InterestedIn: genders, AgeMin: req.AgeMin, AgeMax: req.AgeMax, MaxDistanceKm: req.MaxDistanceKm,
	})
}

func (s *Service) Card(ctx context.Context, viewerID, targetID string) (Card, error) {
	card, err := s.repo.GetCard(ctx, viewerID, targetID)
	if errors.Is(err, ErrNotFound) {
		return Card{}, platformerrors.ErrNotFound
	}
	return card, err
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.ToLower(strings.TrimSpace(v))
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
