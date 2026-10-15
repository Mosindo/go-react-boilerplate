package users

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

var (
	ErrProfileNotFound   = platformerrors.NotFound("profile_not_found", "profile not found")
	ErrInvalidFirstName  = platformerrors.BadRequest("invalid_first_name", "first name must be 1 to 40 characters")
	ErrInvalidBirthDate  = platformerrors.BadRequest("invalid_birth_date", "birth date must use the YYYY-MM-DD format")
	ErrUnderage          = platformerrors.BadRequest("underage", "you must be at least 18 years old")
	ErrInvalidAge        = platformerrors.BadRequest("invalid_birth_date", "birth date is not valid")
	ErrBirthDateLocked   = platformerrors.Conflict("birth_date_locked", "the birth date cannot be changed")
	ErrInvalidGender     = platformerrors.BadRequest("invalid_gender", "gender is not valid")
	ErrInvalidBio        = platformerrors.BadRequest("invalid_bio", "bio must be at most 500 characters")
	ErrInvalidInterests  = platformerrors.BadRequest("invalid_interests", "interests are not valid (10 at most, from the catalog)")
	ErrInvalidPrefs      = platformerrors.BadRequest("invalid_preferences", "preferences are not valid")
	ErrInvalidAgeRange   = platformerrors.BadRequest("invalid_age_range", "age range must be between 18 and 99")
	ErrInvalidDistance   = platformerrors.BadRequest("invalid_distance", "distance must be between 1 and 500 km")
	ErrInvalidLocation   = platformerrors.BadRequest("invalid_location", "location is not valid")
	ErrProfileIncomplete = platformerrors.Conflict("profile_incomplete", "complete your profile, preferences and add a photo first")
)

// locationPrecision rounds coordinates to 2 decimals (about 1 km) before
// storage: precise enough for "people near me", too coarse to locate a home.
const locationPrecision = 100

type Service struct {
	repo   Repository
	photos PhotoSource
	now    func() time.Time
}

func NewService(repo Repository, photos PhotoSource) *Service {
	return &Service{repo: repo, photos: photos, now: time.Now}
}

// ---- reading ----

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

func (s *Service) Onboarding(ctx context.Context, userID string) (Onboarding, error) {
	var o Onboarding
	if _, err := s.repo.GetProfile(ctx, userID); err == nil {
		o.Profile = true
	} else if !errors.Is(err, ErrRepositoryNotFound) {
		return o, err
	}
	if _, err := s.repo.GetPreferences(ctx, userID); err == nil {
		o.Preferences = true
	} else if !errors.Is(err, ErrRepositoryNotFound) {
		return o, err
	}
	n, err := s.repo.PhotoCount(ctx, userID)
	if err != nil {
		return o, err
	}
	o.Photos = n > 0
	o.Complete = o.Profile && o.Preferences && o.Photos
	return o, nil
}

// RequireComplete is used by matching to refuse actions until the profile is usable.
func (s *Service) RequireComplete(ctx context.Context, userID string) error {
	o, err := s.Onboarding(ctx, userID)
	if err != nil {
		return err
	}
	if !o.Complete {
		return ErrProfileIncomplete
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userID string) (MyProfileResponse, error) {
	onboarding, err := s.Onboarding(ctx, userID)
	if err != nil {
		return MyProfileResponse{}, err
	}
	if !onboarding.Profile {
		return MyProfileResponse{Onboarding: onboarding}, nil
	}

	profile, err := s.repo.GetProfile(ctx, userID)
	if err != nil {
		return MyProfileResponse{}, err
	}
	interests, err := s.repo.InterestsForUsers(ctx, []string{userID})
	if err != nil {
		return MyProfileResponse{}, err
	}
	photos, err := s.photos.PhotosForUsers(ctx, []string{userID})
	if err != nil {
		return MyProfileResponse{}, err
	}

	me := &MyProfile{
		FirstName:    profile.FirstName,
		BirthDate:    profile.BirthDate.Format("2006-01-02"),
		Age:          AgeAt(profile.BirthDate, s.now()),
		Gender:       profile.Gender,
		Bio:          profile.Bio,
		City:         profile.City,
		HasLocation:  profile.Latitude != nil,
		IsVisible:    profile.IsVisible,
		ShowDistance: profile.ShowDistance,
		Interests:    orEmptyInterests(interests[userID]),
		Photos:       orEmptyPhotos(photos[userID]),
	}
	if onboarding.Preferences {
		prefs, err := s.repo.GetPreferences(ctx, userID)
		if err != nil {
			return MyProfileResponse{}, err
		}
		me.Preferences = &PreferencesDTO{
			InterestedIn:  prefs.InterestedIn,
			MinAge:        prefs.MinAge,
			MaxAge:        prefs.MaxAge,
			MaxDistanceKm: prefs.MaxDistanceKm,
		}
	}
	return MyProfileResponse{Profile: me, Onboarding: onboarding}, nil
}

// ---- writing ----

func (s *Service) SaveProfile(ctx context.Context, userID string, in ProfileInput) error {
	firstName, err := normalizeFirstName(in.FirstName)
	if err != nil {
		return err
	}
	birthDate, err := time.Parse("2006-01-02", strings.TrimSpace(in.BirthDate))
	if err != nil {
		return ErrInvalidBirthDate
	}
	age := AgeAt(birthDate, s.now())
	if birthDate.After(s.now()) || age > MaxAge {
		return ErrInvalidAge
	}
	if age < MinAge {
		return ErrUnderage
	}
	gender := strings.TrimSpace(in.Gender)
	if _, ok := validGenders[gender]; !ok {
		return ErrInvalidGender
	}
	bio, err := normalizeBio(in.Bio)
	if err != nil {
		return err
	}
	slugs, err := normalizeInterestSlugs(in.Interests)
	if err != nil {
		return err
	}

	existing, err := s.repo.GetProfile(ctx, userID)
	switch {
	case err == nil:
		if !existing.BirthDate.Equal(birthDate) {
			return ErrBirthDateLocked
		}
	case !errors.Is(err, ErrRepositoryNotFound):
		return err
	}

	err = s.repo.SaveProfile(ctx, Profile{
		UserID:    userID,
		FirstName: firstName,
		BirthDate: birthDate,
		Gender:    gender,
		Bio:       bio,
	}, slugs)
	if errors.Is(err, ErrRepositoryUnknownInterest) {
		return ErrInvalidInterests
	}
	return err
}

func (s *Service) SavePreferences(ctx context.Context, userID string, in PreferencesInput) error {
	if len(in.InterestedIn) == 0 || len(in.InterestedIn) > len(validGenders) {
		return ErrInvalidPrefs
	}
	seen := make(map[string]struct{}, len(in.InterestedIn))
	genders := make([]string, 0, len(in.InterestedIn))
	for _, g := range in.InterestedIn {
		g = strings.TrimSpace(g)
		if _, ok := validGenders[g]; !ok {
			return ErrInvalidPrefs
		}
		if _, dup := seen[g]; dup {
			continue
		}
		seen[g] = struct{}{}
		genders = append(genders, g)
	}
	if in.MinAge < MinAge || in.MaxAge > MaxAge || in.MinAge > in.MaxAge {
		return ErrInvalidAgeRange
	}
	if in.MaxDistanceKm < 1 || in.MaxDistanceKm > 500 {
		return ErrInvalidDistance
	}
	if _, err := s.repo.GetProfile(ctx, userID); err != nil {
		if errors.Is(err, ErrRepositoryNotFound) {
			return ErrProfileNotFound
		}
		return err
	}
	return s.repo.SavePreferences(ctx, userID, Preferences{
		InterestedIn:  genders,
		MinAge:        in.MinAge,
		MaxAge:        in.MaxAge,
		MaxDistanceKm: in.MaxDistanceKm,
	})
}

func (s *Service) SetLocation(ctx context.Context, userID string, in LocationInput) error {
	if in.Latitude == nil || in.Longitude == nil {
		return ErrInvalidLocation
	}
	lat, lng := *in.Latitude, *in.Longitude
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return ErrInvalidLocation
	}
	var city *string
	if trimmed := cleanText(in.City); trimmed != "" {
		if utf8.RuneCountInString(trimmed) > MaxCityRunes {
			return ErrInvalidLocation
		}
		city = &trimmed
	}
	err := s.repo.SetLocation(ctx, userID, RoundCoordinate(lat), RoundCoordinate(lng), city)
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrProfileNotFound
	}
	return err
}

func (s *Service) ClearLocation(ctx context.Context, userID string) error {
	err := s.repo.ClearLocation(ctx, userID)
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrProfileNotFound
	}
	return err
}

func (s *Service) SetPrivacy(ctx context.Context, userID string, in PrivacyInput) error {
	err := s.repo.SetPrivacy(ctx, userID, in.IsVisible, in.ShowDistance)
	if errors.Is(err, ErrRepositoryNotFound) {
		return ErrProfileNotFound
	}
	return err
}

func (s *Service) TouchActive(ctx context.Context, userID string) {
	_ = s.repo.TouchActive(ctx, userID)
}

// ---- cards ----

// BuildCards turns query rows into public cards using three batched queries
// (interests of everyone, photos of everyone, interests of the viewer).
func (s *Service) BuildCards(ctx context.Context, viewerID string, bases []CardBase) ([]Card, error) {
	if len(bases) == 0 {
		return []Card{}, nil
	}
	ids := make([]string, 0, len(bases)+1)
	for _, b := range bases {
		ids = append(ids, b.UserID)
	}
	interests, err := s.repo.InterestsForUsers(ctx, append(ids, viewerID))
	if err != nil {
		return nil, err
	}
	photos, err := s.photos.PhotosForUsers(ctx, ids)
	if err != nil {
		return nil, err
	}

	mine := make(map[string]struct{}, len(interests[viewerID]))
	for _, i := range interests[viewerID] {
		mine[i.Slug] = struct{}{}
	}

	now := s.now()
	cards := make([]Card, 0, len(bases))
	for _, b := range bases {
		common := 0
		for _, i := range interests[b.UserID] {
			if _, ok := mine[i.Slug]; ok {
				common++
			}
		}
		card := Card{
			UserID:          b.UserID,
			FirstName:       b.FirstName,
			Age:             AgeAt(b.BirthDate, now),
			Gender:          b.Gender,
			Bio:             b.Bio,
			City:            b.City,
			Interests:       orEmptyInterests(interests[b.UserID]),
			CommonInterests: common,
			Photos:          orEmptyPhotos(photos[b.UserID]),
			LikedMe:         b.LikedMe,
		}
		if b.DistanceKm != nil {
			d := ApproximateDistanceKm(*b.DistanceKm)
			card.DistanceKm = &d
		}
		cards = append(cards, card)
	}
	return cards, nil
}

// ---- pure helpers (unit tested) ----

// AgeAt returns the age in whole years on the given day.
func AgeAt(birth, on time.Time) int {
	years := on.Year() - birth.Year()
	if on.Month() < birth.Month() || (on.Month() == birth.Month() && on.Day() < birth.Day()) {
		years--
	}
	return years
}

// ApproximateDistanceKm rounds up to the next multiple of 5 km (minimum 5)
// so exact positions cannot be recovered by comparing distances.
func ApproximateDistanceKm(km float64) int {
	if km <= 5 {
		return 5
	}
	return int(math.Ceil(km/5.0)) * 5
}

func RoundCoordinate(v float64) float64 {
	return math.Round(v*locationPrecision) / locationPrecision
}

// cleanText trims, collapses runs of spaces and drops control characters.
func cleanText(s string) string {
	var b strings.Builder
	lastSpace := false
	for _, r := range strings.TrimSpace(s) {
		switch {
		case unicode.IsSpace(r):
			if !lastSpace {
				b.WriteRune(' ')
			}
			lastSpace = true
		case unicode.IsControl(r):
		default:
			b.WriteRune(r)
			lastSpace = false
		}
	}
	return b.String()
}

func normalizeFirstName(raw string) (string, error) {
	name := cleanText(raw)
	n := utf8.RuneCountInString(name)
	if n < 1 || n > MaxFirstNameRunes {
		return "", ErrInvalidFirstName
	}
	return name, nil
}

// normalizeBio keeps line breaks (at most one blank line in a row) and drops other control characters.
func normalizeBio(raw string) (string, error) {
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(raw), "\r\n", "\n"), "\n")
	cleaned := make([]string, 0, len(lines))
	blank := 0
	for _, line := range lines {
		line = cleanText(line)
		if line == "" {
			blank++
			if blank > 1 {
				continue
			}
		} else {
			blank = 0
		}
		cleaned = append(cleaned, line)
	}
	bio := strings.Join(cleaned, "\n")
	if utf8.RuneCountInString(bio) > MaxBioRunes {
		return "", ErrInvalidBio
	}
	return bio, nil
}

func normalizeInterestSlugs(raw []string) ([]string, error) {
	if len(raw) > MaxInterests {
		return nil, ErrInvalidInterests
	}
	seen := make(map[string]struct{}, len(raw))
	slugs := make([]string, 0, len(raw))
	for _, slug := range raw {
		slug = strings.ToLower(strings.TrimSpace(slug))
		if slug == "" {
			return nil, ErrInvalidInterests
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		slugs = append(slugs, slug)
	}
	return slugs, nil
}

func orEmptyInterests(v []Interest) []Interest {
	if v == nil {
		return []Interest{}
	}
	return v
}

func orEmptyPhotos(v []PhotoView) []PhotoView {
	if v == nil {
		return []PhotoView{}
	}
	return v
}
