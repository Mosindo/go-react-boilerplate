package profiles

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/httpx"
)

var ErrNotViewable = errors.New("profile not viewable")

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service { return &Service{repo: repo, now: time.Now} }

// Repo exposes batch lookups to sibling features (matching, chat).
func (s *Service) ListPublic(ctx context.Context, viewerID string, ids []string) ([]PublicProfile, error) {
	return s.repo.ListPublic(ctx, viewerID, ids)
}

func (s *Service) GetOwn(ctx context.Context, userID string) (OwnProfile, error) {
	return s.repo.GetOwn(ctx, userID)
}

func (s *Service) GetPublic(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	ok, err := s.repo.Viewable(ctx, viewerID, targetID)
	if err != nil {
		return PublicProfile{}, err
	}
	if !ok {
		return PublicProfile{}, ErrNotViewable
	}
	list, err := s.repo.ListPublic(ctx, viewerID, []string{targetID})
	if err != nil {
		return PublicProfile{}, err
	}
	if len(list) == 0 {
		return PublicProfile{}, ErrNotViewable
	}
	return list[0], nil
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpsertProfileRequest) (OwnProfile, error) {
	firstName, err := cleanText(req.FirstName, maxFirstNameLen, false)
	if err != nil || firstName == "" || !strings.ContainsFunc(firstName, unicode.IsLetter) {
		return OwnProfile{}, httpx.Invalid("first name is required (1-50 characters, with letters)")
	}
	if _, ok := validGenders[req.Gender]; !ok {
		return OwnProfile{}, httpx.Invalid("invalid gender")
	}
	bio, err := cleanText(req.Bio, maxBioLen, true)
	if err != nil {
		return OwnProfile{}, httpx.Invalid("bio must be at most 500 characters")
	}
	city, err := cleanText(req.City, maxCityLen, false)
	if err != nil {
		return OwnProfile{}, httpx.Invalid("city must be at most 80 characters")
	}

	var interests []string
	if req.Interests != nil {
		interests = uniqueSorted(req.Interests)
		if len(interests) > maxInterests {
			return OwnProfile{}, httpx.Invalid("at most 10 interests")
		}
	}

	existing, err := s.repo.GetOwn(ctx, userID)
	hasProfile := err == nil
	if err != nil && !errors.Is(err, ErrProfileNotFound) {
		return OwnProfile{}, err
	}

	var birth time.Time
	if hasProfile {
		birth, _ = time.Parse("2006-01-02", existing.BirthDate)
		if req.BirthDate != "" && req.BirthDate != existing.BirthDate {
			return OwnProfile{}, httpx.Invalid("birth date cannot be changed")
		}
	} else {
		birth, err = time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
		if err != nil {
			return OwnProfile{}, httpx.Invalid("birth date must be formatted YYYY-MM-DD")
		}
		age := AgeOn(birth, s.now())
		if age < MinAge {
			return OwnProfile{}, httpx.Invalid("you must be at least 18 years old")
		}
		if age > MaxAge {
			return OwnProfile{}, httpx.Invalid("invalid birth date")
		}
	}

	err = s.repo.Upsert(ctx, userID, ProfileInput{
		FirstName: firstName, BirthDate: birth, Gender: req.Gender, Bio: bio, City: city,
		Interests: interests, Discoverable: req.Discoverable, ShowDistance: req.ShowDistance,
	})
	if errors.Is(err, ErrUnknownInterest) {
		return OwnProfile{}, httpx.Invalid("unknown interest")
	}
	if err != nil {
		return OwnProfile{}, err
	}
	return s.repo.GetOwn(ctx, userID)
}

func (s *Service) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	return s.repo.GetPreferences(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (Preferences, error) {
	interested := uniqueSorted(req.InterestedIn)
	if len(interested) == 0 {
		return Preferences{}, httpx.Invalid("choose at least one gender to meet")
	}
	for _, g := range interested {
		if _, ok := validGenders[g]; !ok {
			return Preferences{}, httpx.Invalid("invalid gender in interestedIn")
		}
	}
	if req.AgeMin < MinAge || req.AgeMax > MaxAge || req.AgeMin > req.AgeMax {
		return Preferences{}, httpx.Invalid("age range must be within 18-99 and min <= max")
	}
	distance := req.MaxDistanceKm
	if distance == 0 {
		distance = defaultDistance
	}
	if distance < 1 || distance > maxDistanceKm {
		return Preferences{}, httpx.Invalid("max distance must be between 1 and 20000 km")
	}
	p := Preferences{InterestedIn: interested, AgeMin: req.AgeMin, AgeMax: req.AgeMax, MaxDistanceKm: distance}
	if err := s.repo.UpdatePreferences(ctx, userID, p); err != nil {
		return Preferences{}, err
	}
	return p, nil
}

// SetLocation stores coordinates rounded to ~1 km; nil clears them.
func (s *Service) SetLocation(ctx context.Context, userID string, req UpdateLocationRequest) error {
	city, err := cleanText(req.City, maxCityLen, false)
	if err != nil {
		return httpx.Invalid("city must be at most 80 characters")
	}
	if req.Latitude == nil || req.Longitude == nil {
		return httpx.Invalid("latitude and longitude are required")
	}
	lat, lng := *req.Latitude, *req.Longitude
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return httpx.Invalid("coordinates out of range")
	}
	lat, lng = RoundCoordinate(lat), RoundCoordinate(lng)
	return s.repo.SetLocation(ctx, userID, &lat, &lng, city)
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	return s.repo.SetLocation(ctx, userID, nil, nil, "")
}

func (s *Service) ListInterests(ctx context.Context) ([]InterestRef, error) {
	return s.repo.ListInterests(ctx)
}

// RoundCoordinate keeps two decimals (~1.1 km): enough to rank by distance,
// too coarse to locate someone.
func RoundCoordinate(v float64) float64 { return math.Round(v*100) / 100 }

// AgeOn returns the completed years between birth and now.
func AgeOn(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

// cleanText trims, drops control characters and enforces a rune limit.
// Newlines are kept only for multiline fields (bio).
func cleanText(raw string, maxRunes int, multiline bool) (string, error) {
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r == '\n' && multiline:
			b.WriteRune(r)
		case r == '\r':
		case unicode.IsControl(r):
			b.WriteRune(' ')
		default:
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	if !multiline {
		out = strings.Join(strings.Fields(out), " ")
	}
	if utf8.RuneCountInString(out) > maxRunes {
		return "", errors.New("too long")
	}
	return out, nil
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// CanView reports whether viewer may see target's profile and photos.
func (s *Service) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	return s.repo.Viewable(ctx, viewerID, targetID)
}
