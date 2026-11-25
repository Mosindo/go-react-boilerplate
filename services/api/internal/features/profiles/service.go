package profiles

import (
	"context"
	"errors"
	"math"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/features/photos"
	apperr "example.com/api/internal/platform/errors"
)

var (
	ErrProfileNotFound   = apperr.New(http.StatusNotFound, "profile_not_found", "profile not found")
	ErrProfileRequired   = apperr.New(http.StatusConflict, "profile_required", "create your profile first")
	ErrInvalidFirstName  = apperr.Validation("invalid_first_name", "first name must be 1 to 40 letters")
	ErrInvalidBirthdate  = apperr.Validation("invalid_birthdate", "birthdate must be a valid date (YYYY-MM-DD)")
	ErrUnderage          = apperr.Validation("underage", "you must be at least 18 years old to use the app")
	ErrInvalidGender     = apperr.Validation("invalid_gender", "gender must be one of woman, man, nonbinary")
	ErrInvalidBio        = apperr.Validation("invalid_bio", "bio must be at most 500 characters")
	ErrInvalidJobTitle   = apperr.Validation("invalid_job_title", "job title must be at most 60 characters")
	ErrInvalidCity       = apperr.Validation("invalid_city", "city must be at most 60 characters")
	ErrInvalidHeight     = apperr.Validation("invalid_height", "height must be between 120 and 230 cm")
	ErrInvalidLookingFor = apperr.Validation("invalid_looking_for", "invalid relationship goal")
	ErrInvalidInterests  = apperr.Validation("invalid_interests", "choose up to 10 distinct interests from the list")
	ErrInvalidPrefs      = apperr.Validation("invalid_preferences", "invalid preferences: choose at least one gender, ages 18-99 and a distance of 1-500 km")
	ErrInvalidLocation   = apperr.Validation("invalid_location", "invalid coordinates")
)

type PhotoLister interface {
	List(ctx context.Context, userID string) ([]photos.Photo, error)
	ListForUsers(ctx context.Context, userIDs []string) (map[string][]photos.Photo, error)
}

type Service struct {
	repo   Repository
	photos PhotoLister
	now    func() time.Time
}

func NewService(repo Repository, p PhotoLister) *Service {
	return &Service{repo: repo, photos: p, now: time.Now}
}

func (s *Service) Me(ctx context.Context, userID string) (MeProfileResponse, error) {
	resp := MeProfileResponse{Privacy: Privacy{Discoverable: true, ShowDistance: true}}

	prefs, _, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return resp, err
	}
	resp.Preferences = prefs

	list, err := s.photos.List(ctx, userID)
	if err != nil {
		return resp, err
	}
	resp.Photos = list

	stored, err := s.repo.GetProfile(ctx, userID)
	switch {
	case errors.Is(err, errRepoNotFound):
		resp.Completeness = computeCompleteness(nil, prefs, len(list))
		return resp, nil
	case err != nil:
		return resp, err
	}

	interests, err := s.repo.InterestsForUsers(ctx, []string{userID})
	if err != nil {
		return resp, err
	}
	own := OwnProfile{
		FirstName:         stored.FirstName,
		Birthdate:         stored.Birthdate.Format(birthdateLayout),
		Age:               AgeAt(stored.Birthdate, s.now()),
		Gender:            stored.Gender,
		Bio:               stored.Bio,
		JobTitle:          stored.JobTitle,
		City:              stored.City,
		HeightCm:          stored.HeightCm,
		LookingFor:        stored.LookingFor,
		Interests:         nonNil(interests[userID]),
		HasLocation:       stored.Latitude != nil,
		LocationUpdatedAt: stored.LocationUpdatedAt,
	}
	resp.Profile = &own
	resp.Privacy = Privacy{Discoverable: stored.Discoverable, ShowDistance: stored.ShowDistance}
	resp.Completeness = computeCompleteness(&own, prefs, len(list))
	return resp, nil
}

// IsComplete reports whether userID can use discovery.
func (s *Service) IsComplete(ctx context.Context, userID string) (bool, error) {
	me, err := s.Me(ctx, userID)
	if err != nil {
		return false, err
	}
	return me.Completeness.IsComplete, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID string, req UpdateProfileRequest) (MeProfileResponse, error) {
	in, err := s.validateProfile(req)
	if err != nil {
		return MeProfileResponse{}, err
	}
	if err := s.repo.UpsertProfile(ctx, userID, in); err != nil {
		if errors.Is(err, errRepoUnknownInterest) {
			return MeProfileResponse{}, ErrInvalidInterests
		}
		return MeProfileResponse{}, err
	}
	return s.Me(ctx, userID)
}

func (s *Service) UpdatePreferences(ctx context.Context, userID string, req UpdatePreferencesRequest) (MeProfileResponse, error) {
	prefs, err := ValidatePreferences(req)
	if err != nil {
		return MeProfileResponse{}, err
	}
	if err := s.repo.UpsertPreferences(ctx, userID, prefs); err != nil {
		return MeProfileResponse{}, err
	}
	return s.Me(ctx, userID)
}

func (s *Service) UpdatePrivacy(ctx context.Context, userID string, req UpdatePrivacyRequest) (MeProfileResponse, error) {
	err := s.repo.UpdatePrivacy(ctx, userID, Privacy{Discoverable: *req.Discoverable, ShowDistance: *req.ShowDistance})
	if errors.Is(err, errRepoNotFound) {
		return MeProfileResponse{}, ErrProfileRequired
	}
	if err != nil {
		return MeProfileResponse{}, err
	}
	return s.Me(ctx, userID)
}

func (s *Service) UpdateLocation(ctx context.Context, userID string, req UpdateLocationRequest) (MeProfileResponse, error) {
	lat, lon, err := CoarsenLocation(*req.Latitude, *req.Longitude)
	if err != nil {
		return MeProfileResponse{}, err
	}
	err = s.repo.UpdateLocation(ctx, userID, lat, lon)
	if errors.Is(err, errRepoNotFound) {
		return MeProfileResponse{}, ErrProfileRequired
	}
	if err != nil {
		return MeProfileResponse{}, err
	}
	return s.Me(ctx, userID)
}

func (s *Service) Interests(ctx context.Context) ([]Interest, error) {
	return s.repo.ListInterests(ctx)
}

// Public returns the profile of targetID as seen by viewerID.
func (s *Service) Public(ctx context.Context, viewerID, targetID string) (PublicProfile, error) {
	card, err := s.repo.VisibleCard(ctx, viewerID, targetID)
	if errors.Is(err, errRepoNotFound) {
		return PublicProfile{}, ErrProfileNotFound
	}
	if err != nil {
		return PublicProfile{}, err
	}
	cards, err := s.Hydrate(ctx, viewerID, []CardRow{card})
	if err != nil {
		return PublicProfile{}, err
	}
	return cards[0], nil
}

// Hydrate turns raw card rows into public profiles, loading interests and
// photos for every row with one query each (no N+1).
func (s *Service) Hydrate(ctx context.Context, viewerID string, rows []CardRow) ([]PublicProfile, error) {
	ids := make([]string, 0, len(rows)+1)
	for _, r := range rows {
		ids = append(ids, r.UserID)
	}
	interests, err := s.repo.InterestsForUsers(ctx, append(ids, viewerID))
	if err != nil {
		return nil, err
	}
	photoMap, err := s.photos.ListForUsers(ctx, ids)
	if err != nil {
		return nil, err
	}
	mine := make(map[int]struct{})
	for _, i := range interests[viewerID] {
		mine[i.ID] = struct{}{}
	}

	now := s.now()
	out := make([]PublicProfile, 0, len(rows))
	for _, r := range rows {
		p := PublicProfile{
			UserID:            r.UserID,
			FirstName:         r.FirstName,
			Age:               AgeAt(r.Birthdate, now),
			Gender:            r.Gender,
			Bio:               r.Bio,
			JobTitle:          r.JobTitle,
			City:              r.City,
			HeightCm:          r.HeightCm,
			LookingFor:        r.LookingFor,
			Interests:         nonNil(interests[r.UserID]),
			CommonInterestIDs: []int{},
			Photos:            photoMap[r.UserID],
			IsDemo:            r.IsDemo,
		}
		if p.Photos == nil {
			p.Photos = []photos.Photo{}
		}
		if r.ShowDistance && r.DistanceKm != nil && r.UserID != viewerID {
			d := ApproximateDistance(*r.DistanceKm)
			p.DistanceKm = &d
		}
		if r.UserID != viewerID {
			for _, i := range p.Interests {
				if _, ok := mine[i.ID]; ok {
					p.CommonInterestIDs = append(p.CommonInterestIDs, i.ID)
				}
			}
		}
		out = append(out, p)
	}
	return out, nil
}

func (s *Service) validateProfile(req UpdateProfileRequest) (profileInput, error) {
	var in profileInput

	name := collapseSpaces(cleanText(req.FirstName, false))
	if n := utf8.RuneCountInString(name); n < 1 || n > 40 || !isName(name) {
		return in, ErrInvalidFirstName
	}
	in.FirstName = name

	birth, err := time.Parse(birthdateLayout, strings.TrimSpace(req.Birthdate))
	if err != nil {
		return in, ErrInvalidBirthdate
	}
	age := AgeAt(birth, s.now())
	if age < MinAge {
		return in, ErrUnderage
	}
	if age > 120 {
		return in, ErrInvalidBirthdate
	}
	in.Birthdate = birth

	if !slices.Contains(Genders, req.Gender) {
		return in, ErrInvalidGender
	}
	in.Gender = req.Gender

	bio := strings.TrimSpace(cleanText(req.Bio, true))
	if utf8.RuneCountInString(bio) > 500 {
		return in, ErrInvalidBio
	}
	in.Bio = bio

	if in.JobTitle, err = optionalText(req.JobTitle, 60, ErrInvalidJobTitle); err != nil {
		return in, err
	}
	if in.City, err = optionalText(req.City, 60, ErrInvalidCity); err != nil {
		return in, err
	}
	if req.HeightCm != nil && (*req.HeightCm < 120 || *req.HeightCm > 230) {
		return in, ErrInvalidHeight
	}
	in.HeightCm = req.HeightCm

	if req.LookingFor != nil && *req.LookingFor != "" {
		if !slices.Contains(LookingFors, *req.LookingFor) {
			return in, ErrInvalidLookingFor
		}
		in.LookingFor = req.LookingFor
	}

	seen := make(map[int]struct{}, len(req.InterestIDs))
	for _, id := range req.InterestIDs {
		if _, dup := seen[id]; dup || id <= 0 || id > math.MaxInt16 {
			return in, ErrInvalidInterests
		}
		seen[id] = struct{}{}
	}
	if len(req.InterestIDs) > MaxInterests {
		return in, ErrInvalidInterests
	}
	in.InterestIDs = req.InterestIDs
	return in, nil
}

// ValidatePreferences checks and normalises discovery preferences.
func ValidatePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	if len(req.InterestedIn) == 0 {
		return Preferences{}, ErrInvalidPrefs
	}
	seen := map[string]struct{}{}
	genders := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		if !slices.Contains(Genders, g) {
			return Preferences{}, ErrInvalidPrefs
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			genders = append(genders, g)
		}
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, ErrInvalidPrefs
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > MaxDistanceKm {
		return Preferences{}, ErrInvalidPrefs
	}
	return Preferences{InterestedIn: genders, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}

// CoarsenLocation validates coordinates and rounds them to ~1 km so the exact
// position of a member is never stored.
func CoarsenLocation(lat, lon float64) (float64, float64, error) {
	if math.IsNaN(lat) || math.IsNaN(lon) || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return 0, 0, ErrInvalidLocation
	}
	return math.Round(lat*locationDecimals) / locationDecimals, math.Round(lon*locationDecimals) / locationDecimals, nil
}

// ApproximateDistance rounds a distance for display. Distances under 2 km are
// shown as 1 km, then rounded to the nearest km (5 km steps beyond 20 km).
func ApproximateDistance(km float64) int {
	switch {
	case km < 2:
		return 1
	case km < 20:
		return int(math.Round(km))
	default:
		return int(math.Round(km/5) * 5)
	}
}

// AgeAt returns the age in full years at the date now.
func AgeAt(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

func computeCompleteness(p *OwnProfile, prefs Preferences, photoCount int) Completeness {
	missing := []string{}
	if p == nil {
		missing = append(missing, "profile")
	}
	if p == nil || !p.HasLocation {
		missing = append(missing, "location")
	}
	if len(prefs.InterestedIn) == 0 {
		missing = append(missing, "preferences")
	}
	if photoCount == 0 {
		missing = append(missing, "photos")
	}
	return Completeness{IsComplete: len(missing) == 0, Missing: missing}
}

func optionalText(v *string, max int, errInvalid error) (*string, error) {
	if v == nil {
		return nil, nil
	}
	clean := collapseSpaces(cleanText(*v, false))
	if clean == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(clean) > max {
		return nil, errInvalid
	}
	return &clean, nil
}

// cleanText removes control characters (keeping newlines when allowed).
func cleanText(v string, allowNewlines bool) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' && allowNewlines {
			return r
		}
		if unicode.IsControl(r) || r == '\u200b' || r == '\ufeff' {
			return -1
		}
		return r
	}, v)
}

func collapseSpaces(v string) string {
	return strings.Join(strings.Fields(v), " ")
}

func isName(v string) bool {
	for _, r := range v {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' && r != '’' && !unicode.Is(unicode.Mn, r) {
			return false
		}
	}
	return true
}

func nonNil(list []Interest) []Interest {
	if list == nil {
		return []Interest{}
	}
	return list
}
