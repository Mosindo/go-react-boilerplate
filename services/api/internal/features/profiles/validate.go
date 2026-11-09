package profiles

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/geo"
)

const (
	MinAge        = 18
	MaxAge        = 100
	maxInterests  = 10
	MaxDistanceKm = 500
)

var ErrValidation = errors.New("validation error")

type validationError struct{ msg string }

func (e validationError) Error() string { return e.msg }
func (e validationError) Unwrap() error { return ErrValidation }

func invalid(msg string) error { return validationError{msg} }

var genders = map[string]bool{"man": true, "woman": true, "nonbinary": true}

// AgeOn computes completed years at the given date.
func AgeOn(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

func cleanText(s string) string { return strings.TrimSpace(s) }

func hasControlChars(s string, allowNewline bool) bool {
	for _, r := range s {
		if allowNewline && (r == '\n' || r == '\r' || r == '\t') {
			continue
		}
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// NormalizeProfile validates and normalizes a profile payload. now is injected for tests.
func NormalizeProfile(in ProfileInput, now time.Time) (ProfileInput, time.Time, error) {
	in.FirstName = cleanText(in.FirstName)
	n := utf8.RuneCountInString(in.FirstName)
	if n < 1 || n > 40 || hasControlChars(in.FirstName, false) {
		return in, time.Time{}, invalid("firstName must be 1 to 40 characters")
	}
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(in.BirthDate))
	if err != nil {
		return in, time.Time{}, invalid("birthDate must be formatted YYYY-MM-DD")
	}
	age := AgeOn(birth, now)
	if age < MinAge {
		return in, time.Time{}, invalid("you must be at least 18 years old")
	}
	if age > MaxAge {
		return in, time.Time{}, invalid("birthDate is not plausible")
	}
	in.Gender = strings.ToLower(strings.TrimSpace(in.Gender))
	if !genders[in.Gender] {
		return in, time.Time{}, invalid("gender must be man, woman or nonbinary")
	}
	in.Bio = cleanText(in.Bio)
	if utf8.RuneCountInString(in.Bio) > 500 || hasControlChars(in.Bio, true) {
		return in, time.Time{}, invalid("bio must be at most 500 characters")
	}
	in.City = cleanText(in.City)
	if utf8.RuneCountInString(in.City) > 80 || hasControlChars(in.City, false) {
		return in, time.Time{}, invalid("city must be at most 80 characters")
	}
	if len(in.InterestIDs) > maxInterests {
		return in, time.Time{}, invalid("at most 10 interests")
	}
	seen := map[int]bool{}
	for _, id := range in.InterestIDs {
		if id <= 0 || seen[id] {
			return in, time.Time{}, invalid("interestIds must be unique positive ids")
		}
		seen[id] = true
	}
	if (in.Latitude == nil) != (in.Longitude == nil) {
		return in, time.Time{}, invalid("latitude and longitude must be provided together")
	}
	if in.Latitude != nil {
		if !geo.ValidCoords(*in.Latitude, *in.Longitude) {
			return in, time.Time{}, invalid("coordinates out of range")
		}
		lat, lon := geo.RoundCoord(*in.Latitude), geo.RoundCoord(*in.Longitude)
		in.Latitude, in.Longitude = &lat, &lon
	}
	return in, birth, nil
}

// NormalizePreferences validates dating preferences.
func NormalizePreferences(p Preferences) (Preferences, error) {
	seen := map[string]bool{}
	out := make([]string, 0, len(p.InterestedIn))
	for _, g := range p.InterestedIn {
		g = strings.ToLower(strings.TrimSpace(g))
		if !genders[g] {
			return p, invalid("interestedIn values must be man, woman or nonbinary")
		}
		if !seen[g] {
			seen[g] = true
			out = append(out, g)
		}
	}
	if len(out) == 0 {
		return p, invalid("interestedIn is required")
	}
	p.InterestedIn = out
	if p.MinAge < MinAge || p.MaxAge > MaxAge || p.MinAge > p.MaxAge {
		return p, invalid("age range must be within 18-100 with minAge <= maxAge")
	}
	if p.MaxDistanceKm < 1 || p.MaxDistanceKm > MaxDistanceKm {
		return p, invalid("maxDistanceKm must be between 1 and 500")
	}
	return p, nil
}
