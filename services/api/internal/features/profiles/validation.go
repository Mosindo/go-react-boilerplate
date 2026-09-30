package profiles

import (
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	apperr "example.com/api/internal/platform/errors"
)

const birthdateLayout = "2006-01-02"

// AgeOn returns the age in full years at the given date.
func AgeOn(birthdate, on time.Time) int {
	years := on.Year() - birthdate.Year()
	if on.Month() < birthdate.Month() || (on.Month() == birthdate.Month() && on.Day() < birthdate.Day()) {
		years--
	}
	return years
}

// ParseBirthdate validates format and enforces the 18+ access rule.
func ParseBirthdate(raw string, now time.Time) (time.Time, error) {
	birthdate, err := time.Parse(birthdateLayout, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, apperr.Validation("birthdate must use the YYYY-MM-DD format")
	}
	age := AgeOn(birthdate, now)
	if age < MinAge {
		return time.Time{}, apperr.New(422, "underage", "you must be at least 18 years old to use the app")
	}
	if age > MaxAge {
		return time.Time{}, apperr.Validation("birthdate is not valid")
	}
	return birthdate, nil
}

// CleanText trims, collapses control characters and enforces a rune limit.
func CleanText(raw string, maxRunes int, field string, multiline bool) (string, error) {
	value := strings.TrimSpace(raw)
	if !utf8.ValidString(value) {
		return "", apperr.Validation(field + " contains invalid characters")
	}
	value = strings.Map(func(r rune) rune {
		if r == '\n' && multiline {
			return r
		}
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	if multiline {
		for strings.Contains(value, "\n\n\n") {
			value = strings.ReplaceAll(value, "\n\n\n", "\n\n")
		}
	}
	if utf8.RuneCountInString(value) > maxRunes {
		return "", apperr.Validation(field + " is too long")
	}
	return value, nil
}

func ValidateFirstName(raw string) (string, error) {
	name, err := CleanText(raw, 40, "firstName", false)
	if err != nil {
		return "", err
	}
	if name == "" {
		return "", apperr.Validation("firstName is required")
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' && r != '’' {
			return "", apperr.Validation("firstName may only contain letters, spaces, hyphens and apostrophes")
		}
	}
	return name, nil
}

func ValidateGender(raw string) (string, error) {
	if !slices.Contains(Genders, raw) {
		return "", apperr.Validation("gender must be one of woman, man, nonbinary")
	}
	return raw, nil
}

func ValidateRelationshipGoal(raw string) (*string, error) {
	if raw == "" {
		return nil, nil
	}
	if !slices.Contains(RelationshipGoals, raw) {
		return nil, apperr.Validation("relationshipGoal is not valid")
	}
	return &raw, nil
}

func ValidatePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	seen := map[string]bool{}
	interested := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		if !slices.Contains(Genders, g) {
			return Preferences{}, apperr.Validation("interestedIn contains an unknown gender")
		}
		if !seen[g] {
			seen[g] = true
			interested = append(interested, g)
		}
	}
	if len(interested) == 0 {
		return Preferences{}, apperr.Validation("select at least one gender you are interested in")
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, apperr.Validation("age range must be within 18-99 and min <= max")
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > MaxDistanceKmLimit {
		return Preferences{}, apperr.Validation("maxDistanceKm must be between 1 and 500")
	}
	return Preferences{
		InterestedIn:  interested,
		MinAge:        req.MinAge,
		MaxAge:        req.MaxAge,
		MaxDistanceKm: req.MaxDistanceKm,
	}, nil
}

// ComputeCompleteness lists what is still needed before appearing in discovery.
func ComputeCompleteness(p storedProfile, photoCount int) Completeness {
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
	if len(p.Preferences.InterestedIn) == 0 {
		missing = append(missing, "interestedIn")
	}
	if photoCount == 0 {
		missing = append(missing, "photos")
	}
	return Completeness{Complete: len(missing) == 0, Missing: missing}
}
