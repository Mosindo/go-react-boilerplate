package profiles

import (
	"errors"
	"math"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrProfileNotFound      = errors.New("profile not found")
	ErrInvalidFirstName     = errors.New("first name must be 1-40 letters")
	ErrInvalidBirthdate     = errors.New("birthdate must be a valid date (YYYY-MM-DD)")
	ErrUnderage             = errors.New("you must be at least 18 years old")
	ErrInvalidAge           = errors.New("birthdate is not plausible")
	ErrBirthdateLocked      = errors.New("birthdate cannot be changed once set")
	ErrInvalidGender        = errors.New("invalid gender")
	ErrInvalidGoal          = errors.New("invalid relationship goal")
	ErrBioTooLong           = errors.New("bio is too long (500 characters max)")
	ErrOccupationTooLong    = errors.New("occupation is too long (80 characters max)")
	ErrCityTooLong          = errors.New("city is too long (80 characters max)")
	ErrInvalidInterestedIn  = errors.New("choose at least one valid gender you are interested in")
	ErrInvalidAgeRange      = errors.New("age range must be within 18-99 and min <= max")
	ErrInvalidDistance      = errors.New("distance must be between 1 and 500 km")
	ErrTooManyInterests     = errors.New("you can choose up to 10 interests")
	ErrUnknownInterest      = errors.New("unknown interest")
	ErrInvalidCoordinates   = errors.New("invalid coordinates")
	ErrProfileNotAccessible = errors.New("profile not accessible")
)

// AgeOn returns the age in full years of someone born on birthdate at date now.
func AgeOn(birthdate, now time.Time) int {
	y1, m1, d1 := birthdate.Date()
	y2, m2, d2 := now.Date()
	age := y2 - y1
	if m2 < m1 || (m2 == m1 && d2 < d1) {
		age--
	}
	return age
}

// BirthdateBounds returns the inclusive birthdate interval matching ages [minAge, maxAge] at now.
func BirthdateBounds(minAge, maxAge int, now time.Time) (earliest, latest time.Time) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	latest = today.AddDate(-minAge, 0, 0)
	earliest = today.AddDate(-(maxAge + 1), 0, 1)
	return earliest, latest
}

// RoundCoordinate keeps 2 decimals (~1.1 km): enough to compute an
// approximate distance, too coarse to locate a home.
func RoundCoordinate(v float64) float64 {
	return math.Round(v*100) / 100
}

// ApproxDistanceKm buckets a distance so precise positions cannot be inferred.
func ApproxDistanceKm(km float64) int {
	switch {
	case km < 2:
		return 1
	case km <= 10:
		return int(math.Round(km))
	default:
		return int(math.Round(km/5) * 5)
	}
}

func normalizeText(v string) string {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, "\r\n", "\n")
	// Collapse runs of blank lines so a bio cannot be used to push content off-screen.
	for strings.Contains(v, "\n\n\n") {
		v = strings.ReplaceAll(v, "\n\n\n", "\n\n")
	}
	return strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, v)
}

func validFirstName(v string) bool {
	n := utf8.RuneCountInString(v)
	if n == 0 || n > maxFirstNameRunes {
		return false
	}
	for _, r := range v {
		if !(unicode.IsLetter(r) || r == ' ' || r == '-' || r == '\'' || r == '’') {
			return false
		}
	}
	return true
}

// ValidateProfile normalizes and validates profile input at time now.
func ValidateProfile(req UpsertProfileRequest, now time.Time) (ProfileInput, error) {
	in := ProfileInput{
		FirstName:        strings.Join(strings.Fields(req.FirstName), " "),
		Gender:           strings.TrimSpace(req.Gender),
		Bio:              normalizeText(req.Bio),
		Occupation:       strings.Join(strings.Fields(req.Occupation), " "),
		RelationshipGoal: strings.TrimSpace(req.RelationshipGoal),
		City:             strings.Join(strings.Fields(req.City), " "),
	}
	if !validFirstName(in.FirstName) {
		return ProfileInput{}, ErrInvalidFirstName
	}
	birthdate, err := time.Parse("2006-01-02", strings.TrimSpace(req.Birthdate))
	if err != nil {
		return ProfileInput{}, ErrInvalidBirthdate
	}
	in.Birthdate = birthdate
	age := AgeOn(birthdate, now)
	if age < MinAge {
		return ProfileInput{}, ErrUnderage
	}
	if age > 120 {
		return ProfileInput{}, ErrInvalidAge
	}
	if !slices.Contains(Genders, in.Gender) {
		return ProfileInput{}, ErrInvalidGender
	}
	if in.RelationshipGoal == "" {
		in.RelationshipGoal = "unsure"
	}
	if !slices.Contains(RelationshipGoals, in.RelationshipGoal) {
		return ProfileInput{}, ErrInvalidGoal
	}
	if utf8.RuneCountInString(in.Bio) > maxBioRunes {
		return ProfileInput{}, ErrBioTooLong
	}
	if utf8.RuneCountInString(in.Occupation) > maxOccupationRune {
		return ProfileInput{}, ErrOccupationTooLong
	}
	if utf8.RuneCountInString(in.City) > maxCityRunes {
		return ProfileInput{}, ErrCityTooLong
	}
	return in, nil
}

// ValidatePreferences normalizes and validates discovery preferences.
func ValidatePreferences(req PreferencesRequest) (Preferences, error) {
	seen := map[string]bool{}
	interested := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		g = strings.TrimSpace(g)
		if !slices.Contains(Genders, g) {
			return Preferences{}, ErrInvalidInterestedIn
		}
		if !seen[g] {
			seen[g] = true
			interested = append(interested, g)
		}
	}
	if len(interested) == 0 {
		return Preferences{}, ErrInvalidInterestedIn
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, ErrInvalidAgeRange
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > 500 {
		return Preferences{}, ErrInvalidDistance
	}
	slices.Sort(interested)
	return Preferences{InterestedIn: interested, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}

// ValidateInterestIDs deduplicates and bounds the interest selection.
func ValidateInterestIDs(ids []int) ([]int, error) {
	out := make([]int, 0, len(ids))
	seen := map[int]bool{}
	for _, id := range ids {
		if id <= 0 || id > math.MaxInt16 {
			return nil, ErrUnknownInterest
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	if len(out) > MaxInterests {
		return nil, ErrTooManyInterests
	}
	return out, nil
}

// ValidateCoordinates checks ranges and rounds to the stored precision.
func ValidateCoordinates(lat, lng float64) (float64, float64, error) {
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, ErrInvalidCoordinates
	}
	return RoundCoordinate(lat), RoundCoordinate(lng), nil
}

// missingSteps lists what prevents a profile from appearing in discovery.
func missingSteps(prefs Preferences, photoCount int) []string {
	missing := []string{}
	if len(prefs.InterestedIn) == 0 {
		missing = append(missing, "preferences")
	}
	if photoCount == 0 {
		missing = append(missing, "photos")
	}
	return missing
}
