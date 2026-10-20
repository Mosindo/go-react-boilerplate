package profiles

import (
	"context"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID guards path parameters before they reach SQL.
func IsUUID(value string) bool { return uuidPattern.MatchString(value) }

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

func (s *Service) ListInterests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) GetMine(ctx context.Context, userID string) (MyProfileResponse, error) {
	profiles, err := s.repo.ListByIDs(ctx, userID, []string{userID})
	if err != nil {
		return MyProfileResponse{}, err
	}
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return MyProfileResponse{}, err
	}
	complete, err := s.repo.IsComplete(ctx, userID)
	if err != nil {
		return MyProfileResponse{}, err
	}
	res := MyProfileResponse{Preferences: prefs, Complete: complete}
	if len(profiles) == 1 {
		res.Profile = &profiles[0]
	}
	return res, nil
}

func (s *Service) UpsertMine(ctx context.Context, userID string, in ProfileInput) (Profile, error) {
	stored, slugs, err := s.validateProfile(in)
	if err != nil {
		return Profile{}, err
	}
	if err := s.repo.UpsertProfile(ctx, userID, stored, slugs); err != nil {
		return Profile{}, err
	}
	profiles, err := s.repo.ListByIDs(ctx, userID, []string{userID})
	if err != nil || len(profiles) != 1 {
		return Profile{}, err
	}
	return profiles[0], nil
}

func (s *Service) validateProfile(in ProfileInput) (storedProfile, []string, error) {
	name := strings.TrimSpace(in.FirstName)
	if n := utf8.RuneCountInString(name); n < 1 || n > 50 {
		return storedProfile{}, nil, platformerrors.BadRequest("first name must be 1 to 50 characters")
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' && r != '’' {
			return storedProfile{}, nil, platformerrors.BadRequest("first name may only contain letters, spaces, hyphens and apostrophes")
		}
	}

	birth, err := time.Parse("2006-01-02", strings.TrimSpace(in.BirthDate))
	if err != nil {
		return storedProfile{}, nil, platformerrors.BadRequest("birth date must use the YYYY-MM-DD format")
	}
	age := AgeOn(birth, s.now())
	if age < MinAge {
		return storedProfile{}, nil, platformerrors.Forbidden("you must be at least 18 years old to use this app")
	}
	if age > MaxAge {
		return storedProfile{}, nil, platformerrors.BadRequest("birth date is not plausible")
	}

	if _, ok := validGenders[in.Gender]; !ok {
		return storedProfile{}, nil, platformerrors.BadRequest("gender must be man, woman or non_binary")
	}

	bio := strings.TrimSpace(in.Bio)
	if utf8.RuneCountInString(bio) > 500 {
		return storedProfile{}, nil, platformerrors.BadRequest("bio must be at most 500 characters")
	}
	city := strings.TrimSpace(in.City)
	if utf8.RuneCountInString(city) > 80 {
		return storedProfile{}, nil, platformerrors.BadRequest("city must be at most 80 characters")
	}

	seen := make(map[string]struct{}, len(in.Interests))
	slugs := make([]string, 0, len(in.Interests))
	for _, slug := range in.Interests {
		slug = strings.TrimSpace(slug)
		if _, dup := seen[slug]; dup || slug == "" {
			continue
		}
		seen[slug] = struct{}{}
		slugs = append(slugs, slug)
	}
	if len(slugs) > MaxInterests {
		return storedProfile{}, nil, platformerrors.BadRequest("at most 10 interests are allowed")
	}

	return storedProfile{FirstName: name, BirthDate: birth, Gender: in.Gender, Bio: bio, City: city}, slugs, nil
}

// AgeOn returns the completed years between birth and now.
func AgeOn(birth, now time.Time) int {
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	return age
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, in Preferences) (Preferences, error) {
	if in.MinAge > in.MaxAge {
		return Preferences{}, platformerrors.BadRequest("minimum age cannot exceed maximum age")
	}
	seen := map[string]struct{}{}
	genders := make([]string, 0, 3)
	for _, g := range in.InterestedIn {
		if _, ok := validGenders[g]; !ok {
			return Preferences{}, platformerrors.BadRequest("invalid gender preference")
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			genders = append(genders, g)
		}
	}
	in.InterestedIn = genders
	if err := s.repo.UpsertPreferences(ctx, userID, in); err != nil {
		return Preferences{}, err
	}
	return in, nil
}

// UpdateLocation keeps only ~1 km of precision (2 decimals).
func (s *Service) UpdateLocation(ctx context.Context, userID string, lat, lng float64) error {
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return platformerrors.BadRequest("invalid coordinates")
	}
	lat = math.Round(lat*100) / 100
	lng = math.Round(lng*100) / 100
	updated, err := s.repo.UpdateLocation(ctx, userID, lat, lng)
	if err != nil {
		return err
	}
	if !updated {
		return platformerrors.Conflict("create your profile before setting a location")
	}
	return nil
}

// GetVisible returns another user's full profile when the caller may see it.
func (s *Service) GetVisible(ctx context.Context, viewerID, targetID string) (Profile, error) {
	if !IsUUID(targetID) {
		return Profile{}, platformerrors.NotFound("profile not found")
	}
	if viewerID != targetID {
		ok, err := s.repo.CanView(ctx, viewerID, targetID)
		if err != nil {
			return Profile{}, err
		}
		if !ok {
			return Profile{}, platformerrors.NotFound("profile not found")
		}
	}
	profiles, err := s.repo.ListByIDs(ctx, viewerID, []string{targetID})
	if err != nil {
		return Profile{}, err
	}
	if len(profiles) != 1 {
		return Profile{}, platformerrors.NotFound("profile not found")
	}
	return profiles[0], nil
}

// CanView is shared with the photos feature.
func (s *Service) CanView(ctx context.Context, viewerID, targetID string) (bool, error) {
	if viewerID == targetID {
		return true, nil
	}
	return s.repo.CanView(ctx, viewerID, targetID)
}

// Hydrate loads full profiles for ids, in the same order, for discovery.
func (s *Service) Hydrate(ctx context.Context, viewerID string, ids []string) ([]Profile, error) {
	return s.repo.ListByIDs(ctx, viewerID, ids)
}
