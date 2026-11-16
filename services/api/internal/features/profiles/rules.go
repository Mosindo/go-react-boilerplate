package profiles

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// ValidationError carries a user-facing message for a 400 response.
type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }

func invalid(format string, args ...any) error {
	return ValidationError{Message: fmt.Sprintf(format, args...)}
}

func IsValidationError(err error) bool {
	var v ValidationError
	return errors.As(err, &v)
}

// AgeOn returns the age in full years at the given instant.
func AgeOn(birthdate, now time.Time) int {
	y1, m1, d1 := birthdate.Date()
	y2, m2, d2 := now.Date()
	age := y2 - y1
	if m2 < m1 || (m2 == m1 && d2 < d1) {
		age--
	}
	return age
}

func cleanText(s string) string {
	s = strings.TrimSpace(s)
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

func cleanLine(s string) string {
	return strings.Join(strings.Fields(cleanText(s)), " ")
}

func validateProfile(in ProfileInput, now time.Time) (profileRecord, error) {
	rec := profileRecord{
		FirstName:  cleanLine(in.FirstName),
		Gender:     strings.TrimSpace(in.Gender),
		Bio:        cleanText(in.Bio),
		JobTitle:   cleanLine(in.JobTitle),
		LookingFor: strings.TrimSpace(in.LookingFor),
		City:       cleanLine(in.City),
		HeightCm:   in.HeightCm,
	}
	if n := utf8.RuneCountInString(rec.FirstName); n < 1 || n > MaxNameLength {
		return rec, invalid("first name must be between 1 and %d characters", MaxNameLength)
	}
	for _, r := range rec.FirstName {
		if unicode.IsDigit(r) || strings.ContainsRune("@/\\<>{}[]_=+*#$%^&|~`", r) {
			return rec, invalid("first name contains invalid characters")
		}
	}
	birth, err := time.Parse("2006-01-02", strings.TrimSpace(in.Birthdate))
	if err != nil {
		return rec, invalid("birthdate must use the YYYY-MM-DD format")
	}
	age := AgeOn(birth, now)
	if age < MinAge {
		return rec, invalid("you must be at least %d years old", MinAge)
	}
	if age > MaxAge {
		return rec, invalid("birthdate is not plausible")
	}
	rec.Birthdate = birth
	if !slices.Contains(Genders, rec.Gender) {
		return rec, invalid("gender must be one of %s", strings.Join(Genders, ", "))
	}
	if rec.LookingFor == "" {
		rec.LookingFor = "unsure"
	}
	if !slices.Contains(LookingFor, rec.LookingFor) {
		return rec, invalid("lookingFor must be one of %s", strings.Join(LookingFor, ", "))
	}
	if utf8.RuneCountInString(rec.Bio) > MaxBioLength {
		return rec, invalid("bio must be at most %d characters", MaxBioLength)
	}
	if utf8.RuneCountInString(rec.JobTitle) > MaxJobLength {
		return rec, invalid("job title must be at most %d characters", MaxJobLength)
	}
	if utf8.RuneCountInString(rec.City) > MaxCityLength {
		return rec, invalid("city must be at most %d characters", MaxCityLength)
	}
	if rec.HeightCm != nil && (*rec.HeightCm < 120 || *rec.HeightCm > 230) {
		return rec, invalid("height must be between 120 and 230 cm")
	}
	seen := map[int]bool{}
	for _, id := range in.InterestIDs {
		if id <= 0 {
			return rec, invalid("invalid interest id")
		}
		if !seen[id] {
			seen[id] = true
			rec.InterestIDs = append(rec.InterestIDs, id)
		}
	}
	if len(rec.InterestIDs) > MaxInterests {
		return rec, invalid("choose at most %d interests", MaxInterests)
	}
	return rec, nil
}

func validatePreferences(in PreferencesInput) (Preferences, error) {
	out := Preferences{AgeMin: in.AgeMin, AgeMax: in.AgeMax, MaxDistanceKm: in.MaxDistanceKm}
	seen := map[string]bool{}
	for _, g := range in.InterestedIn {
		g = strings.TrimSpace(g)
		if !slices.Contains(Genders, g) {
			return out, invalid("interestedIn must only contain %s", strings.Join(Genders, ", "))
		}
		if !seen[g] {
			seen[g] = true
			out.InterestedIn = append(out.InterestedIn, g)
		}
	}
	if len(out.InterestedIn) == 0 {
		return out, invalid("choose at least one gender you are interested in")
	}
	slices.Sort(out.InterestedIn)
	if out.AgeMin < MinAge || out.AgeMax > MaxAge || out.AgeMin > out.AgeMax {
		return out, invalid("age range must be within %d-%d and min <= max", MinAge, MaxAge)
	}
	if out.MaxDistanceKm < MinDistanceKm || out.MaxDistanceKm > MaxDistanceKm {
		return out, invalid("distance must be between %d and %d km", MinDistanceKm, MaxDistanceKm)
	}
	return out, nil
}

// CoarsenCoordinate rounds to 2 decimals (about 1.1 km of latitude). Exact
// positions are never stored.
func CoarsenCoordinate(v float64) float64 {
	return math.Round(v*100) / 100
}

func validateLocation(lat, lng float64) (float64, float64, error) {
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, invalid("invalid coordinates")
	}
	return CoarsenCoordinate(lat), CoarsenCoordinate(lng), nil
}

// ApproximateDistanceKm converts a raw distance into a deliberately coarse
// value suitable for display (1 km minimum, 5 km steps beyond 10 km).
func ApproximateDistanceKm(km float64) int {
	switch {
	case km < 1.5:
		return 1
	case km < 10:
		return int(math.Round(km))
	default:
		return int(math.Round(km/5) * 5)
	}
}

// HaversineKm returns the great-circle distance between two points.
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusKm * math.Asin(math.Min(1, math.Sqrt(a)))
}
