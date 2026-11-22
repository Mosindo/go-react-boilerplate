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
	ErrInvalidFirstName   = errors.New("first name must be 1-40 letters")
	ErrInvalidBirthdate   = errors.New("birthdate must be a valid date (YYYY-MM-DD)")
	ErrUnderage           = errors.New("you must be at least 18 years old")
	ErrBirthdateLocked    = errors.New("birthdate cannot be changed once set")
	ErrInvalidGender      = errors.New("invalid gender")
	ErrBioTooLong         = errors.New("bio is too long")
	ErrJobTitleTooLong    = errors.New("job title is too long")
	ErrTooManyInterests   = errors.New("too many interests")
	ErrUnknownInterest    = errors.New("unknown interest")
	ErrInvalidLocation    = errors.New("invalid coordinates")
	ErrCityTooLong        = errors.New("city is too long")
	ErrInvalidPreferences = errors.New("invalid preferences")
	ErrProfileNotFound    = errors.New("profile not found")
)

// NormalizeFirstName trims and validates a first name. Letters, spaces,
// hyphens and apostrophes are allowed (e.g. "Anne-Sophie", "D'Arcy").
func NormalizeFirstName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	n := utf8.RuneCountInString(name)
	if n == 0 || n > MaxFirstNameRunes {
		return "", ErrInvalidFirstName
	}
	hasLetter := false
	for _, r := range name {
		switch {
		case unicode.IsLetter(r) || unicode.Is(unicode.Mn, r):
			hasLetter = true
		case r == ' ' || r == '-' || r == '\'' || r == '’':
		default:
			return "", ErrInvalidFirstName
		}
	}
	if !hasLetter {
		return "", ErrInvalidFirstName
	}
	return name, nil
}

// ParseBirthdate validates format and the 18+ access rule.
func ParseBirthdate(raw string, now time.Time) (time.Time, error) {
	d, err := time.Parse(birthdateLayout, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, ErrInvalidBirthdate
	}
	age := AgeAt(d, now)
	if d.After(now) || age > 100 {
		return time.Time{}, ErrInvalidBirthdate
	}
	if age < MinAge {
		return time.Time{}, ErrUnderage
	}
	return d, nil
}

// AgeAt returns completed years between birthdate and now.
func AgeAt(birthdate, now time.Time) int {
	y1, m1, d1 := birthdate.Date()
	y2, m2, d2 := now.Date()
	age := y2 - y1
	if m2 < m1 || (m2 == m1 && d2 < d1) {
		age--
	}
	return age
}

func ValidGender(g string) bool {
	return slices.Contains(AllGenders, g)
}

// NormalizeText trims and bounds free text, collapsing runs of blank lines.
func NormalizeText(raw string, max int, tooLong error) (string, error) {
	text := strings.TrimSpace(strings.ReplaceAll(raw, "\r\n", "\n"))
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}
	if utf8.RuneCountInString(text) > max {
		return "", tooLong
	}
	return text, nil
}

// NormalizeInterestSlugs de-duplicates and bounds the requested interests.
func NormalizeInterestSlugs(slugs []string) ([]string, error) {
	out := make([]string, 0, len(slugs))
	for _, s := range slugs {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" || len(s) > 40 {
			return nil, ErrUnknownInterest
		}
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	if len(out) > MaxInterests {
		return nil, ErrTooManyInterests
	}
	return out, nil
}

// RoundCoordinates snaps a position onto a ~1 km grid so the exact location
// of a member is never stored.
func RoundCoordinates(lat, lng float64) (float64, float64, error) {
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) ||
		lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, ErrInvalidLocation
	}
	return math.Round(lat*coordinatePrecision) / coordinatePrecision,
		math.Round(lng*coordinatePrecision) / coordinatePrecision, nil
}

// DisplayDistanceKm turns a raw distance into a coarse public value
// (whole kilometres, never below 1) so nobody can be pinpointed.
func DisplayDistanceKm(km float64) int {
	if km < 1 {
		return 1
	}
	if km < 10 {
		return int(math.Round(km))
	}
	return int(math.Round(km/5) * 5)
}

// DistanceKm is the haversine distance used for ranking and tests.
func DistanceKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0088
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(a))
}

// ValidatePreferences enforces coherent discovery settings.
func ValidatePreferences(req PreferencesRequest) (Preferences, error) {
	genders := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		g = strings.ToLower(strings.TrimSpace(g))
		if !ValidGender(g) {
			return Preferences{}, ErrInvalidPreferences
		}
		if !slices.Contains(genders, g) {
			genders = append(genders, g)
		}
	}
	if len(genders) == 0 ||
		req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge ||
		req.MaxDistanceKm < MinDistanceKm || req.MaxDistanceKm > MaxDistanceKm {
		return Preferences{}, ErrInvalidPreferences
	}
	slices.Sort(genders)
	return Preferences{InterestedIn: genders, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}

// missingFields lists what still blocks a profile from being discoverable.
func missingFields(p storedProfile) []string {
	missing := make([]string, 0, 6)
	if p.FirstName == nil || *p.FirstName == "" {
		missing = append(missing, "firstName")
	}
	if p.Birthdate == nil {
		missing = append(missing, "birthdate")
	}
	if p.Gender == nil || *p.Gender == "" {
		missing = append(missing, "gender")
	}
	if len(p.Preferences.InterestedIn) == 0 {
		missing = append(missing, "interestedIn")
	}
	if p.Latitude == nil || p.Longitude == nil {
		missing = append(missing, "location")
	}
	if p.PhotoCount == 0 {
		missing = append(missing, "photos")
	}
	return missing
}
