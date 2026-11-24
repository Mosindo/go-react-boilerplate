package profiles

import (
	"math"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	apperr "example.com/api/internal/platform/errors"
)

var (
	ErrInvalidFirstName  = apperr.BadRequest("invalid_first_name", "first name must be 1 to 40 letters")
	ErrInvalidBirthdate  = apperr.BadRequest("invalid_birthdate", "birthdate must be a valid date (YYYY-MM-DD)")
	ErrUnderage          = apperr.BadRequest("underage", "you must be at least 18 years old to use Lueur")
	ErrInvalidGender     = apperr.BadRequest("invalid_gender", "gender must be woman, man or nonbinary")
	ErrInvalidGoal       = apperr.BadRequest("invalid_relationship_goal", "unknown relationship goal")
	ErrBioTooLong        = apperr.BadRequest("bio_too_long", "bio must be 500 characters or less")
	ErrJobTooLong        = apperr.BadRequest("job_title_too_long", "job title must be 80 characters or less")
	ErrCityTooLong       = apperr.BadRequest("city_too_long", "city must be 80 characters or less")
	ErrInvalidHeight     = apperr.BadRequest("invalid_height", "height must be between 120 and 230 cm")
	ErrInvalidLocation   = apperr.BadRequest("invalid_location", "latitude/longitude are out of range")
	ErrInvalidAgeRange   = apperr.BadRequest("invalid_age_range", "age range must be within 18-99 and min <= max")
	ErrInvalidDistance   = apperr.BadRequest("invalid_distance", "distance must be between 1 and 300 km")
	ErrInvalidInterested = apperr.BadRequest("invalid_interested_in", "interestedIn must list woman, man and/or nonbinary")
	ErrTooManyInterests  = apperr.BadRequest("too_many_interests", "choose up to 10 interests")
	ErrUnknownInterest   = apperr.BadRequest("unknown_interest", "one of the interests does not exist")
)

// AgeAt returns the age in full years at the given instant.
func AgeAt(birthdate, now time.Time) int {
	now = now.UTC()
	age := now.Year() - birthdate.Year()
	if now.Month() < birthdate.Month() || (now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
		age--
	}
	return age
}

func cleanText(value string) string {
	value = strings.TrimSpace(value)
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, value)
}

func validName(name string) bool {
	n := utf8.RuneCountInString(name)
	if n < 1 || n > 40 {
		return false
	}
	for _, r := range name {
		if !(unicode.IsLetter(r) || r == ' ' || r == '-' || r == '\'' || r == '’') {
			return false
		}
	}
	return true
}

// NormalizeProfile validates and cleans a profile submission.
func NormalizeProfile(req UpsertProfileRequest, now time.Time) (Profile, error) {
	p := Profile{
		FirstName:        strings.Join(strings.Fields(req.FirstName), " "),
		Gender:           strings.TrimSpace(req.Gender),
		Bio:              cleanText(req.Bio),
		JobTitle:         strings.Join(strings.Fields(req.JobTitle), " "),
		HeightCm:         req.HeightCm,
		RelationshipGoal: strings.TrimSpace(req.RelationshipGoal),
		City:             strings.Join(strings.Fields(req.City), " "),
	}
	if !validName(p.FirstName) {
		return Profile{}, ErrInvalidFirstName
	}
	birthdate, err := time.Parse("2006-01-02", strings.TrimSpace(req.Birthdate))
	if err != nil || birthdate.After(now) {
		return Profile{}, ErrInvalidBirthdate
	}
	age := AgeAt(birthdate, now)
	if age < MinAge {
		return Profile{}, ErrUnderage
	}
	if age > 120 {
		return Profile{}, ErrInvalidBirthdate
	}
	p.Birthdate = birthdate
	if !slices.Contains(Genders, p.Gender) {
		return Profile{}, ErrInvalidGender
	}
	if p.RelationshipGoal == "" {
		p.RelationshipGoal = "unsure"
	}
	if !slices.Contains(RelationshipGoals, p.RelationshipGoal) {
		return Profile{}, ErrInvalidGoal
	}
	if utf8.RuneCountInString(p.Bio) > 500 {
		return Profile{}, ErrBioTooLong
	}
	if utf8.RuneCountInString(p.JobTitle) > 80 {
		return Profile{}, ErrJobTooLong
	}
	if utf8.RuneCountInString(p.City) > 80 {
		return Profile{}, ErrCityTooLong
	}
	if p.HeightCm != nil && (*p.HeightCm < 120 || *p.HeightCm > 230) {
		return Profile{}, ErrInvalidHeight
	}
	return p, nil
}

// RoundCoordinate snaps a coordinate to a 0.01° grid (~1.1 km), so the exact
// position of a member is never stored.
func RoundCoordinate(v float64) float64 {
	return math.Round(v*100) / 100
}

func NormalizeLocation(lat, lng float64) (float64, float64, error) {
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, ErrInvalidLocation
	}
	return RoundCoordinate(lat), RoundCoordinate(lng), nil
}

// ApproximateDistance converts a raw distance into what members see: whole
// kilometres, never below 1 km, so co-located members can't be pinpointed.
func ApproximateDistance(km float64) int {
	if km < 1 {
		return 1
	}
	if km < 10 {
		return int(math.Round(km))
	}
	return int(math.Round(km/5) * 5)
}

func NormalizePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	seen := map[string]bool{}
	interested := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		g = strings.TrimSpace(g)
		if !slices.Contains(Genders, g) {
			return Preferences{}, ErrInvalidInterested
		}
		if !seen[g] {
			seen[g] = true
			interested = append(interested, g)
		}
	}
	if len(interested) == 0 {
		return Preferences{}, ErrInvalidInterested
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, ErrInvalidAgeRange
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > 300 {
		return Preferences{}, ErrInvalidDistance
	}
	return Preferences{InterestedIn: interested, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}

func NormalizeInterestIDs(ids []int) ([]int, error) {
	seen := map[int]bool{}
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return nil, ErrUnknownInterest
		}
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	if len(out) > MaxInterestsPerUser {
		return nil, ErrTooManyInterests
	}
	return out, nil
}
