package profiles

import (
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"example.com/api/internal/platform/people"
)

var (
	ErrInvalidFirstName   = errors.New("first name must be 1 to 30 letters")
	ErrInvalidBirthdate   = errors.New("birthdate must be a valid YYYY-MM-DD date")
	ErrUnderage           = errors.New("you must be at least 18 years old")
	ErrInvalidAge         = errors.New("birthdate is not plausible")
	ErrBirthdateLocked    = errors.New("birthdate cannot be changed once set")
	ErrInvalidGender      = errors.New("invalid gender")
	ErrInvalidGoal        = errors.New("invalid relationship goal")
	ErrInvalidHeight      = errors.New("height must be between 120 and 230 cm")
	ErrTextTooLong        = errors.New("text field too long")
	ErrInvalidText        = errors.New("text contains forbidden characters")
	ErrInvalidPreferences = errors.New("invalid preferences")
	ErrInvalidLocation    = errors.New("invalid location")
	ErrTooManyInterests   = errors.New("at most 10 interests")
	ErrUnknownInterest    = errors.New("unknown interest")
	ErrProfileRequired    = errors.New("create your profile first")
)

const (
	maxFirstNameRunes = 30
	maxBioRunes       = 500
	maxJobTitleRunes  = 60
	maxCityRunes      = 60
	MaxInterests      = 10
)

var firstNamePattern = regexp.MustCompile(`^[\p{L}][\p{L}\p{M}'’ -]*$`)

func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func validateFirstName(raw string) (string, error) {
	name := normalizeSpaces(raw)
	if name == "" || utf8.RuneCountInString(name) > maxFirstNameRunes || !firstNamePattern.MatchString(name) {
		return "", ErrInvalidFirstName
	}
	return name, nil
}

// validateBirthdate parses the date and enforces the 18+ access rule.
func validateBirthdate(raw string, now time.Time) (time.Time, error) {
	birthdate, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, ErrInvalidBirthdate
	}
	age := people.Age(birthdate, now)
	if age < people.MinimumAge {
		return time.Time{}, ErrUnderage
	}
	if age > people.MaximumAge+21 {
		return time.Time{}, ErrInvalidAge
	}
	return birthdate, nil
}

// cleanText trims, rejects control characters (except new lines when
// multiline) and enforces a rune budget.
func cleanText(raw string, maxRunes int, multiline bool) (string, error) {
	text := strings.TrimSpace(raw)
	if !multiline {
		text = normalizeSpaces(text)
	}
	if utf8.RuneCountInString(text) > maxRunes {
		return "", ErrTextTooLong
	}
	for _, r := range text {
		if r == '\n' && multiline {
			continue
		}
		if unicode.IsControl(r) {
			return "", ErrInvalidText
		}
	}
	if multiline {
		for strings.Contains(text, "\n\n\n") {
			text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
		}
	}
	return text, nil
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func validatePreferences(req PreferencesRequest) (Preferences, error) {
	seen := map[string]struct{}{}
	interested := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		g = strings.TrimSpace(g)
		if !contains(Genders, g) {
			return Preferences{}, ErrInvalidPreferences
		}
		if _, dup := seen[g]; dup {
			continue
		}
		seen[g] = struct{}{}
		interested = append(interested, g)
	}
	if len(interested) == 0 {
		return Preferences{}, ErrInvalidPreferences
	}
	if req.MinAge < people.MinimumAge || req.MaxAge > people.MaximumAge || req.MinAge > req.MaxAge {
		return Preferences{}, ErrInvalidPreferences
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > 500 {
		return Preferences{}, ErrInvalidPreferences
	}
	return Preferences{InterestedIn: interested, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}

func validateLocation(lat, lng float64) error {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 || (lat == 0 && lng == 0) {
		return ErrInvalidLocation
	}
	return nil
}
