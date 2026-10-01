package profiles

import (
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrInvalidName        = errors.New("invalid first name")
	ErrInvalidBirthDate   = errors.New("invalid birth date")
	ErrUnderage           = errors.New("must be at least 18 years old")
	ErrInvalidGender      = errors.New("invalid gender")
	ErrBioTooLong         = errors.New("bio too long")
	ErrCityTooLong        = errors.New("city too long")
	ErrInvalidPreferences = errors.New("invalid preferences")
	ErrInvalidLocation    = errors.New("invalid location")
	ErrInvalidInterests   = errors.New("invalid interests")
	ErrProfileMissing     = errors.New("profile not created")
)

// AgeOn returns the age in whole years of someone born on birth, as of now (UTC dates).
func AgeOn(birth, now time.Time) int {
	by, bm, bd := birth.UTC().Date()
	ny, nm, nd := now.UTC().Date()
	age := ny - by
	if nm < bm || (nm == bm && nd < bd) {
		age--
	}
	return age
}

// ValidateBirthDate enforces the 18+ rule on the server; clients are never trusted for it.
func ValidateBirthDate(birth, now time.Time) error {
	if birth.After(now) {
		return ErrInvalidBirthDate
	}
	age := AgeOn(birth, now)
	if age < MinAge {
		return ErrUnderage
	}
	if age > MaxAge {
		return ErrInvalidBirthDate
	}
	return nil
}

func ValidGender(g string) bool {
	for _, v := range Genders {
		if v == g {
			return true
		}
	}
	return false
}

// CleanText trims, collapses nothing else, and rejects control characters other than newline.
func CleanText(s string) string {
	s = strings.TrimSpace(s)
	return strings.Map(func(r rune) rune {
		if r == '\n' || r >= 0x20 && r != 0x7f {
			return r
		}
		return -1
	}, s)
}

func ValidateProfileText(name, bio, city string) error {
	if n := utf8.RuneCountInString(name); n < 1 || n > MaxNameRunes {
		return ErrInvalidName
	}
	if utf8.RuneCountInString(bio) > MaxBioRunes {
		return ErrBioTooLong
	}
	if utf8.RuneCountInString(city) > MaxCityRunes {
		return ErrCityTooLong
	}
	return nil
}

func ValidatePreferences(p Preferences) error {
	if len(p.InterestedIn) == 0 || len(p.InterestedIn) > len(Genders) {
		return ErrInvalidPreferences
	}
	seen := map[string]bool{}
	for _, g := range p.InterestedIn {
		if !ValidGender(g) || seen[g] {
			return ErrInvalidPreferences
		}
		seen[g] = true
	}
	if p.MinAge < MinAge || p.MaxAge > MaxAge || p.MinAge > p.MaxAge {
		return ErrInvalidPreferences
	}
	if p.MaxDistanceKm < 1 || p.MaxDistanceKm > 500 {
		return ErrInvalidPreferences
	}
	return nil
}

// SnapCoordinate rounds a coordinate to 2 decimals (~1.1 km). Only the snapped value is stored,
// so the database never holds a user's exact position.
func SnapCoordinate(v float64) float64 { return math.Round(v*100) / 100 }

func ValidateLocation(lat, lng float64) error {
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) ||
		lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return ErrInvalidLocation
	}
	return nil
}

// distanceStepKm is the granularity of the distance shown to other users.
const distanceStepKm = 5

// BucketDistanceKm turns an exact distance into an approximate one (rounded up to 5 km steps,
// minimum 5), which blocks trilateration of a user's position.
func BucketDistanceKm(km float64) int {
	if km <= distanceStepKm {
		return distanceStepKm
	}
	return int(math.Ceil(km/distanceStepKm)) * distanceStepKm
}

// MissingFields lists what is still needed before a profile can appear in discovery.
func MissingFields(exists bool, photoCount int) []string {
	var missing []string
	if !exists {
		missing = append(missing, "profile")
	}
	if photoCount == 0 {
		missing = append(missing, "photos")
	}
	return missing
}
