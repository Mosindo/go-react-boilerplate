package profiles

import (
	"errors"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalidFirstName   = errors.New("invalid first name")
	ErrInvalidBirthdate   = errors.New("invalid birthdate")
	ErrUnderage           = errors.New("you must be at least 18 years old")
	ErrInvalidGender      = errors.New("invalid gender")
	ErrBioTooLong         = errors.New("bio too long")
	ErrInvalidCity        = errors.New("invalid city")
	ErrInvalidPreferences = errors.New("invalid preferences")
	ErrInvalidLocation    = errors.New("invalid location")
	ErrInvalidInterests   = errors.New("invalid interests")
)

// AgeOn returns the whole years between birthdate and now.
func AgeOn(birthdate, now time.Time) int {
	years := now.Year() - birthdate.Year()
	if now.Month() < birthdate.Month() || (now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
		years--
	}
	return years
}

// ApproxDistanceKm turns an exact distance into a coarse bucket (5 km steps)
// so repeated queries from different spots cannot triangulate someone.
func ApproxDistanceKm(distance float64) int {
	if math.IsNaN(distance) || distance < 0 {
		return 5
	}
	if distance <= 5 {
		return 5
	}
	return int(math.Ceil(distance/5)) * 5
}

// RoundCoordinate keeps two decimals (about 1.1 km): the most precise
// location the server ever stores.
func RoundCoordinate(v float64) float64 {
	return math.Round(v*100) / 100
}

func normalizeFirstName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	if name == "" || utf8.RuneCountInString(name) > MaxFirstName {
		return "", ErrInvalidFirstName
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' && r != '’' && r != '.' {
			return "", ErrInvalidFirstName
		}
	}
	return name, nil
}

func parseBirthdate(raw string, now time.Time) (time.Time, error) {
	birthdate, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, ErrInvalidBirthdate
	}
	age := AgeOn(birthdate, now)
	if age < 0 || age > MaxAge {
		return time.Time{}, ErrInvalidBirthdate
	}
	if age < MinAge {
		return time.Time{}, ErrUnderage
	}
	return birthdate, nil
}

func normalizeBio(raw string) (string, error) {
	var b strings.Builder
	for _, r := range strings.TrimSpace(raw) {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			b.WriteRune(r)
		}
	}
	bio := strings.TrimSpace(b.String())
	if utf8.RuneCountInString(bio) > MaxBioRunes {
		return "", ErrBioTooLong
	}
	return bio, nil
}

func normalizeCity(raw string) (string, error) {
	city := strings.Join(strings.Fields(raw), " ")
	if utf8.RuneCountInString(city) > MaxCityRunes {
		return "", ErrInvalidCity
	}
	for _, r := range city {
		if unicode.IsControl(r) {
			return "", ErrInvalidCity
		}
	}
	return city, nil
}

func normalizePreferences(p Preferences) (Preferences, error) {
	if p.MinAge < MinAge || p.MaxAge > MaxAge || p.MinAge > p.MaxAge {
		return Preferences{}, ErrInvalidPreferences
	}
	if p.MaxDistanceKm < 1 || p.MaxDistanceKm > MaxDistanceKm {
		return Preferences{}, ErrInvalidPreferences
	}
	seen := map[string]struct{}{}
	genders := make([]string, 0, 3)
	for _, g := range p.InterestedIn {
		if _, ok := validGenders[g]; !ok {
			return Preferences{}, ErrInvalidPreferences
		}
		if _, dup := seen[g]; dup {
			continue
		}
		seen[g] = struct{}{}
		genders = append(genders, g)
	}
	if len(genders) == 0 {
		return Preferences{}, ErrInvalidPreferences
	}
	p.InterestedIn = genders
	return p, nil
}

func normalizeLocation(lat, lng float64, city string) (float64, float64, string, error) {
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) ||
		lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, "", ErrInvalidLocation
	}
	cleanCity, err := normalizeCity(city)
	if err != nil {
		return 0, 0, "", err
	}
	return RoundCoordinate(lat), RoundCoordinate(lng), cleanCity, nil
}
