package profiles

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) GetOwn(ctx context.Context, userID string) (Profile, error) {
	p, err := s.repo.GetProfile(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return Profile{}, platformerrors.NotFound("profile not created yet")
	}
	return p, err
}

func (s *Service) Upsert(ctx context.Context, userID string, req UpdateProfileRequest) (Profile, error) {
	name := strings.Join(strings.Fields(req.FirstName), " ")
	if n := utf8.RuneCountInString(name); n < 1 || n > MaxNameLength {
		return Profile{}, platformerrors.Invalid("first name must be between 1 and %d characters", MaxNameLength)
	}
	if containsControl(name) {
		return Profile{}, platformerrors.Invalid("first name contains invalid characters")
	}

	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return Profile{}, platformerrors.Invalid("birth date must use the YYYY-MM-DD format")
	}
	age := AgeAt(birth, s.now())
	if age < MinAge {
		return Profile{}, platformerrors.Forbidden("you must be at least %d years old to use this app", MinAge)
	}
	if age > 120 {
		return Profile{}, platformerrors.Invalid("birth date is not valid")
	}

	gender := strings.TrimSpace(req.Gender)
	if _, ok := validGenders[gender]; !ok {
		return Profile{}, platformerrors.Invalid("gender must be one of man, woman, non_binary")
	}

	bio := strings.TrimSpace(req.Bio)
	if utf8.RuneCountInString(bio) > MaxBioLength {
		return Profile{}, platformerrors.Invalid("bio must be at most %d characters", MaxBioLength)
	}
	city := strings.Join(strings.Fields(req.City), " ")
	if utf8.RuneCountInString(city) > MaxCityLength {
		return Profile{}, platformerrors.Invalid("city must be at most %d characters", MaxCityLength)
	}

	var lat, lng *float64
	if (req.Latitude == nil) != (req.Longitude == nil) {
		return Profile{}, platformerrors.Invalid("latitude and longitude must be provided together")
	}
	if req.Latitude != nil {
		la, lo := *req.Latitude, *req.Longitude
		if math.IsNaN(la) || math.IsNaN(lo) || la < -90 || la > 90 || lo < -180 || lo > 180 {
			return Profile{}, platformerrors.Invalid("coordinates are out of range")
		}
		// Only ~1 km precision is ever stored.
		la, lo = roundCoord(la), roundCoord(lo)
		lat, lng = &la, &lo
	}

	if len(req.InterestIDs) > MaxInterests {
		return Profile{}, platformerrors.Invalid("you can pick at most %d interests", MaxInterests)
	}
	interestIDs := dedupe(req.InterestIDs)

	visible, showDistance := true, true
	if existing, err := s.repo.GetProfile(ctx, userID); err == nil {
		visible, showDistance = existing.IsVisible, existing.ShowDistance
	} else if !errors.Is(err, ErrNotFound) {
		return Profile{}, err
	}
	if req.IsVisible != nil {
		visible = *req.IsVisible
	}
	if req.ShowDistance != nil {
		showDistance = *req.ShowDistance
	}

	p := Profile{
		UserID: userID, FirstName: name, BirthDate: birth, Gender: gender, Bio: bio, City: city,
		IsVisible: visible, ShowDistance: showDistance,
	}
	if err := s.repo.UpsertProfile(ctx, p, lat, lng, interestIDs); err != nil {
		if errors.Is(err, errUnknownInterest) {
			return Profile{}, platformerrors.Invalid("unknown interest")
		}
		return Profile{}, err
	}
	return s.repo.GetProfile(ctx, userID)
}

func (s *Service) GetPreferences(ctx context.Context, userID string) (Preferences, error) {
	p, err := s.repo.GetPreferences(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return Preferences{}, platformerrors.NotFound("preferences not set yet")
	}
	return p, err
}

func (s *Service) SetPreferences(ctx context.Context, userID string, req PreferencesRequest) (Preferences, error) {
	seen := map[string]struct{}{}
	genders := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		g = strings.TrimSpace(g)
		if _, ok := validGenders[g]; !ok {
			return Preferences{}, platformerrors.Invalid("interestedIn values must be man, woman or non_binary")
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			genders = append(genders, g)
		}
	}
	if len(genders) == 0 {
		return Preferences{}, platformerrors.Invalid("choose at least one gender you are interested in")
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, platformerrors.Invalid("age range must be within %d-%d with min <= max", MinAge, MaxAge)
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > 500 {
		return Preferences{}, platformerrors.Invalid("max distance must be between 1 and 500 km")
	}
	p := Preferences{InterestedIn: genders, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}
	if err := s.repo.UpsertPreferences(ctx, userID, p); err != nil {
		return Preferences{}, err
	}
	return p, nil
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) View(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	p, err := s.repo.GetPublicProfile(ctx, viewerID, targetID)
	if errors.Is(err, ErrNotFound) {
		return PublicProfile{}, platformerrors.NotFound("profile not found")
	}
	return p, err
}

func (s *Service) Now() time.Time { return s.now() }

func dedupe(ids []int16) []int16 {
	seen := make(map[int16]struct{}, len(ids))
	out := make([]int16, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

func containsControl(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}
