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
	ErrInvalidFirstName   = errors.New("invalid first name")
	ErrInvalidBirthdate   = errors.New("invalid birthdate")
	ErrUnderage           = errors.New("user must be at least 18")
	ErrInvalidGender      = errors.New("invalid gender")
	ErrBioTooLong         = errors.New("bio too long")
	ErrJobTitleTooLong    = errors.New("job title too long")
	ErrCityTooLong        = errors.New("city too long")
	ErrTooManyInterests   = errors.New("too many interests")
	ErrUnknownInterest    = errors.New("unknown interest")
	ErrInvalidLocation    = errors.New("invalid location")
	ErrInvalidPreferences = errors.New("invalid preferences")
	ErrProfileNotFound    = errors.New("profile not found")
	ErrProfileIncomplete  = errors.New("profile incomplete")
)

// AgeOn returns the age in full years at the given date.
func AgeOn(birthdate, on time.Time) int {
	years := on.Year() - birthdate.Year()
	if on.Month() < birthdate.Month() || (on.Month() == birthdate.Month() && on.Day() < birthdate.Day()) {
		years--
	}
	return years
}

// ParseBirthdate validates an ISO date and enforces the 18+ access rule.
func ParseBirthdate(raw string, now time.Time) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, ErrInvalidBirthdate
	}
	age := AgeOn(parsed, now)
	if age < MinAge {
		return time.Time{}, ErrUnderage
	}
	if age > MaxAge+20 {
		return time.Time{}, ErrInvalidBirthdate
	}
	return parsed, nil
}

// NormalizeFirstName trims, collapses whitespace and only accepts letters, spaces, hyphens and apostrophes.
func NormalizeFirstName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	count := utf8.RuneCountInString(name)
	if count < 1 || count > MaxFirstNameRunes {
		return "", ErrInvalidFirstName
	}
	for _, r := range name {
		if !(unicode.IsLetter(r) || unicode.IsMark(r) || r == ' ' || r == '-' || r == '\'' || r == '’') {
			return "", ErrInvalidFirstName
		}
	}
	return name, nil
}

func ValidGender(g string) bool {
	return slices.Contains(Genders, g)
}

// NormalizeFreeText trims, strips control characters (except newlines) and limits newlines.
func NormalizeFreeText(raw string, maxRunes int, tooLong error) (string, error) {
	var b strings.Builder
	newlines := 0
	for _, r := range strings.TrimSpace(raw) {
		if r == '\n' {
			newlines++
			if newlines > 8 {
				continue
			}
			b.WriteRune(r)
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	text := strings.TrimSpace(b.String())
	if utf8.RuneCountInString(text) > maxRunes {
		return "", tooLong
	}
	return text, nil
}

// RoundCoordinate reduces precision to ~1 km so exact positions are never stored.
func RoundCoordinate(v float64) float64 {
	return math.Round(v*locationPrecision) / locationPrecision
}

func ValidateCoordinates(lat, lng float64) error {
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return ErrInvalidLocation
	}
	return nil
}

// HaversineKm returns the great-circle distance between two points in kilometres.
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}

// ApproximateDistanceKm rounds a distance for display: never below 1 km, 5 km buckets beyond 20 km.
func ApproximateDistanceKm(km float64) int {
	switch {
	case km < 1:
		return 1
	case km < 20:
		return int(math.Round(km))
	default:
		return int(math.Round(km/5) * 5)
	}
}

func ValidatePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	if len(req.InterestedIn) < 1 || len(req.InterestedIn) > len(Genders) {
		return Preferences{}, ErrInvalidPreferences
	}
	seen := map[string]bool{}
	interested := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		g = strings.TrimSpace(g)
		if !ValidGender(g) || seen[g] {
			return Preferences{}, ErrInvalidPreferences
		}
		seen[g] = true
		interested = append(interested, g)
	}
	if req.AgeMin < MinAge || req.AgeMax > MaxAge || req.AgeMin > req.AgeMax {
		return Preferences{}, ErrInvalidPreferences
	}
	if req.MaxDistanceKm < MinDistanceKm || req.MaxDistanceKm > MaxDistanceKm {
		return Preferences{}, ErrInvalidPreferences
	}
	return Preferences{InterestedIn: interested, AgeMin: req.AgeMin, AgeMax: req.AgeMax, MaxDistanceKm: req.MaxDistanceKm}, nil
}

// MissingSteps lists onboarding steps that still block discovery.
func MissingSteps(p StoredProfile, photoCount int) []string {
	missing := []string{}
	if p.FirstName == "" {
		missing = append(missing, "firstName")
	}
	if p.Birthdate == nil {
		missing = append(missing, "birthdate")
	}
	if p.Gender == nil {
		missing = append(missing, "gender")
	}
	if photoCount == 0 {
		missing = append(missing, "photos")
	}
	if p.Latitude == nil || p.Longitude == nil {
		missing = append(missing, "location")
	}
	return missing
}
