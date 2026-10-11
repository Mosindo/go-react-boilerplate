package users

import (
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
)

var (
	ErrProfileNotFound = platformerrors.New(http.StatusNotFound, "profile_not_found", "profile not found")
	ErrNotFound        = platformerrors.New(http.StatusNotFound, "not_found", "not found")
	ErrUnderage        = platformerrors.New(http.StatusBadRequest, "underage", "you must be at least 18 years old")
	ErrBirthDateLocked = platformerrors.New(http.StatusBadRequest, "invalid_request", "birth date cannot be changed")
	ErrUnauthorized    = platformerrors.New(http.StatusUnauthorized, "unauthorized", "account not found")
)

func invalid(msg string) *platformerrors.AppError {
	return platformerrors.New(http.StatusBadRequest, "invalid_request", msg)
}

const maxInterests = 10

// normalizeText trims, drops control characters (keeping newlines in multi-line fields) and collapses nothing else.
func normalizeText(s string, multiline bool) string {
	s = strings.Map(func(r rune) rune {
		if r == '\n' && multiline {
			return r
		}
		if r == '\r' || unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}

// ValidateProfile checks and normalises profile input. `now` is injected for deterministic age checks.
func ValidateProfile(in ProfileInput, now time.Time) (ProfileInput, time.Time, error) {
	in.FirstName = normalizeText(in.FirstName, false)
	in.Bio = normalizeText(in.Bio, true)
	in.City = normalizeText(in.City, false)
	in.Gender = strings.TrimSpace(in.Gender)

	if n := utf8.RuneCountInString(in.FirstName); n < 1 || n > 40 {
		return in, time.Time{}, invalid("firstName must be 1 to 40 characters")
	}
	for _, r := range in.FirstName {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' && r != '’' {
			return in, time.Time{}, invalid("firstName contains invalid characters")
		}
	}
	if utf8.RuneCountInString(in.Bio) > 500 {
		return in, time.Time{}, invalid("bio must be at most 500 characters")
	}
	if utf8.RuneCountInString(in.City) > 80 {
		return in, time.Time{}, invalid("city must be at most 80 characters")
	}
	if !validGenders[in.Gender] {
		return in, time.Time{}, invalid("gender must be man, woman or non_binary")
	}
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(in.BirthDate))
	if err != nil {
		return in, time.Time{}, invalid("birthDate must be formatted YYYY-MM-DD")
	}
	age := AgeOn(birth, now)
	if age < MinAge {
		return in, time.Time{}, ErrUnderage
	}
	if age > 120 {
		return in, time.Time{}, invalid("birthDate is not plausible")
	}
	if len(in.InterestIDs) > maxInterests {
		return in, time.Time{}, invalid("at most 10 interests")
	}
	seen := make(map[int]bool, len(in.InterestIDs))
	ids := make([]int, 0, len(in.InterestIDs))
	for _, id := range in.InterestIDs {
		if id < 1 {
			return in, time.Time{}, invalid("invalid interest id")
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	in.InterestIDs = ids
	return in, birth, nil
}

// ValidatePreferences checks the dating preferences.
func ValidatePreferences(p Preferences) (Preferences, error) {
	if len(p.InterestedIn) < 1 || len(p.InterestedIn) > 3 {
		return p, invalid("interestedIn needs 1 to 3 genders")
	}
	seen := map[string]bool{}
	uniq := make([]string, 0, 3)
	for _, g := range p.InterestedIn {
		if !validGenders[g] {
			return p, invalid("interestedIn contains an unknown gender")
		}
		if !seen[g] {
			seen[g] = true
			uniq = append(uniq, g)
		}
	}
	p.InterestedIn = uniq
	if p.MinAge < MinAge || p.MaxAge > MaxAge || p.MinAge > p.MaxAge {
		return p, invalid("age range must satisfy 18 <= minAge <= maxAge <= 99")
	}
	if p.MaxDistanceKm != nil && (*p.MaxDistanceKm < 1 || *p.MaxDistanceKm > 500) {
		return p, invalid("maxDistanceKm must be between 1 and 500, or null")
	}
	return p, nil
}

// ApproxDistanceKm coarsens a distance so exact positions can't be triangulated.
func ApproxDistanceKm(d float64) int {
	switch {
	case d <= 1:
		return 1
	case d <= 10:
		return int(d + 0.999)
	case d <= 50:
		return (int(d+4.999) / 5) * 5
	default:
		return (int(d+9.999) / 10) * 10
	}
}

// RoundCoordinate keeps ~1 km precision (0.01 degree) before anything is stored.
func RoundCoordinate(v float64) float64 {
	return float64(int64(v*100+sign(v)*0.5)) / 100
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}
