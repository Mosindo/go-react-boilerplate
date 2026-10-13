package profiles

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// ValidationError carries a message that is safe to show to the client.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func invalid(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}

func IsValidation(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}

// AgeOn returns the age in whole years of someone born on birthDate at the given day.
func AgeOn(birthDate, today time.Time) int {
	years := today.Year() - birthDate.Year()
	if today.Month() < birthDate.Month() || (today.Month() == birthDate.Month() && today.Day() < birthDate.Day()) {
		years--
	}
	return years
}

func cleanText(s string, keepNewlines bool) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\n' && keepNewlines:
			b.WriteRune(r)
		case r == '\r' || r == '\t':
			b.WriteRune(' ')
		case unicode.IsControl(r) || unicode.Is(unicode.Cf, r):
			// dropped: control and invisible formatting characters
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

func ValidateName(raw string) (string, error) {
	name := strings.Join(strings.Fields(cleanText(raw, false)), " ")
	if name == "" {
		return "", invalid("first name is required")
	}
	if utf8.RuneCountInString(name) > MaxNameRunes {
		return "", invalid("first name must be at most %d characters", MaxNameRunes)
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' && r != '’' && r != '.' {
			return "", invalid("first name contains unsupported characters")
		}
	}
	return name, nil
}

func ValidateBirthDate(raw string, now time.Time) (time.Time, error) {
	d, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, invalid("birth date must use the YYYY-MM-DD format")
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	age := AgeOn(d, today)
	if age < MinAge {
		return time.Time{}, invalid("you must be at least %d years old", MinAge)
	}
	if age > 120 {
		return time.Time{}, invalid("birth date is not valid")
	}
	return d, nil
}

func ValidateGender(g string) (string, error) {
	g = strings.ToLower(strings.TrimSpace(g))
	for _, allowed := range Genders {
		if g == allowed {
			return g, nil
		}
	}
	return "", invalid("gender must be one of: woman, man, nonbinary")
}

func ValidateBio(raw string) (string, error) {
	bio := cleanText(raw, true)
	if utf8.RuneCountInString(bio) > MaxBioRunes {
		return "", invalid("bio must be at most %d characters", MaxBioRunes)
	}
	return bio, nil
}

func ValidateCity(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	city := strings.Join(strings.Fields(cleanText(*raw, false)), " ")
	if city == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(city) > MaxCityRunes {
		return nil, invalid("city must be at most %d characters", MaxCityRunes)
	}
	return &city, nil
}

func ValidateInterestIDs(ids []int) ([]int, error) {
	seen := make(map[int]struct{}, len(ids))
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if id < 1 || id > math.MaxInt16 {
			return nil, invalid("unknown interest")
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) > MaxInterests {
		return nil, invalid("choose at most %d interests", MaxInterests)
	}
	return out, nil
}

func ValidatePreferences(req PreferencesRequest) (Preferences, error) {
	seen := map[string]bool{}
	var in []string
	for _, g := range req.InterestedIn {
		g, err := ValidateGender(g)
		if err != nil {
			return Preferences{}, invalid("interestedIn must contain woman, man and/or nonbinary")
		}
		if !seen[g] {
			seen[g] = true
			in = append(in, g)
		}
	}
	if len(in) == 0 {
		return Preferences{}, invalid("choose at least one gender you are interested in")
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, invalid("age range must be between %d and %d", MinAge, MaxAge)
	}
	if req.MaxDistanceKm != nil && (*req.MaxDistanceKm < 1 || *req.MaxDistanceKm > MaxDistanceKm) {
		return Preferences{}, invalid("distance must be between 1 and %d km", MaxDistanceKm)
	}
	return Preferences{InterestedIn: in, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}

// RoundCoordinate reduces precision to ~1 km so exact positions are never stored.
func RoundCoordinate(v float64) float64 {
	return math.Round(v*LocationDecimal) / LocationDecimal
}

func ValidateLocation(req LocationRequest) (lat, lon *float64, err error) {
	if req.Latitude == nil && req.Longitude == nil {
		return nil, nil, nil
	}
	if req.Latitude == nil || req.Longitude == nil {
		return nil, nil, invalid("latitude and longitude must be provided together")
	}
	la, lo := *req.Latitude, *req.Longitude
	if math.IsNaN(la) || math.IsNaN(lo) || la < -90 || la > 90 || lo < -180 || lo > 180 {
		return nil, nil, invalid("coordinates are out of range")
	}
	la, lo = RoundCoordinate(la), RoundCoordinate(lo)
	return &la, &lo, nil
}

// DistanceBucket turns an exact distance into the coarse value shown to other
// users: rounded up to the next 5 km, never below 5.
func DistanceBucket(km float64) int {
	if km <= 5 {
		return 5
	}
	return int(math.Ceil(km/5.0)) * 5
}
