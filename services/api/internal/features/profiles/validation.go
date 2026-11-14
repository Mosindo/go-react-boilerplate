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

// ValidationError carries a stable machine code and a user-facing message.
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string { return e.Code + ": " + e.Message }

func invalid(code, message string) error { return &ValidationError{Code: code, Message: message} }

// AgeAt returns the age in whole years of someone born on birthdate at date now.
func AgeAt(birthdate, now time.Time) int {
	years := now.Year() - birthdate.Year()
	if now.Month() < birthdate.Month() || (now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
		years--
	}
	return years
}

// cleanText trims, collapses control characters and enforces a rune limit.
func cleanText(value string, maxRunes int, multiline bool) (string, bool) {
	value = strings.TrimSpace(value)
	var b strings.Builder
	for _, r := range value {
		if r == '\n' && multiline {
			b.WriteRune(r)
			continue
		}
		if unicode.IsControl(r) {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	cleaned := strings.TrimSpace(b.String())
	if multiline {
		for strings.Contains(cleaned, "\n\n\n") {
			cleaned = strings.ReplaceAll(cleaned, "\n\n\n", "\n\n")
		}
	}
	return cleaned, utf8.RuneCountInString(cleaned) <= maxRunes
}

func ValidateProfile(req UpdateProfileRequest, now time.Time) (ProfileInput, error) {
	firstName, ok := cleanText(req.FirstName, maxFirstNameRunes, false)
	if firstName == "" || !ok {
		return ProfileInput{}, invalid("invalid_first_name", "le prénom doit contenir entre 1 et 40 caractères")
	}
	for _, r := range firstName {
		if !(unicode.IsLetter(r) || r == ' ' || r == '-' || r == '\'' || r == '’') {
			return ProfileInput{}, invalid("invalid_first_name", "le prénom ne peut contenir que des lettres, espaces, tirets et apostrophes")
		}
	}

	birthdate, err := time.Parse(birthdateLayout, strings.TrimSpace(req.Birthdate))
	if err != nil {
		return ProfileInput{}, invalid("invalid_birthdate", "date de naissance invalide (format AAAA-MM-JJ)")
	}
	age := AgeAt(birthdate, now)
	if age < MinAge {
		return ProfileInput{}, invalid("underage", "Lueur est réservée aux personnes majeures (18 ans et plus)")
	}
	if age > MaxAge {
		return ProfileInput{}, invalid("invalid_birthdate", "date de naissance invalide")
	}

	if !slices.Contains(Genders, req.Gender) {
		return ProfileInput{}, invalid("invalid_gender", "genre invalide")
	}

	lookingFor := req.LookingFor
	if lookingFor == "" {
		lookingFor = "not_sure"
	}
	if !slices.Contains(LookingFors, lookingFor) {
		return ProfileInput{}, invalid("invalid_looking_for", "type de relation invalide")
	}

	bio, ok := cleanText(req.Bio, maxBioRunes, true)
	if !ok {
		return ProfileInput{}, invalid("invalid_bio", "la bio est limitée à 500 caractères")
	}
	jobTitle, ok := cleanText(req.JobTitle, maxJobTitleRunes, false)
	if !ok {
		return ProfileInput{}, invalid("invalid_job_title", "la profession est limitée à 80 caractères")
	}
	if req.HeightCm != nil && (*req.HeightCm < minHeightCm || *req.HeightCm > maxHeightCm) {
		return ProfileInput{}, invalid("invalid_height", "taille invalide")
	}

	interests := make([]string, 0, len(req.Interests))
	for _, slug := range req.Interests {
		slug = strings.TrimSpace(slug)
		if slug == "" || len(slug) > 40 {
			return ProfileInput{}, invalid("invalid_interests", "centre d'intérêt invalide")
		}
		if !slices.Contains(interests, slug) {
			interests = append(interests, slug)
		}
	}
	if len(interests) > MaxInterests {
		return ProfileInput{}, invalid("invalid_interests", "10 centres d'intérêt maximum")
	}

	return ProfileInput{
		FirstName:  firstName,
		Birthdate:  birthdate,
		Gender:     req.Gender,
		Bio:        bio,
		JobTitle:   jobTitle,
		HeightCm:   req.HeightCm,
		LookingFor: lookingFor,
		Interests:  interests,
	}, nil
}

func ValidatePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	seen := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		if !slices.Contains(Genders, g) {
			return Preferences{}, invalid("invalid_interested_in", "préférence de genre invalide")
		}
		if !slices.Contains(seen, g) {
			seen = append(seen, g)
		}
	}
	if len(seen) == 0 {
		return Preferences{}, invalid("invalid_interested_in", "choisissez au moins un genre")
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, invalid("invalid_age_range", "tranche d'âge invalide (18 à 99 ans)")
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > MaxDistanceKm {
		return Preferences{}, invalid("invalid_distance", "distance invalide (1 à 500 km)")
	}
	slices.Sort(seen)
	return Preferences{InterestedIn: seen, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}

var errLocation = errors.New("invalid location")

// CoarsenLocation validates coordinates and rounds them to ~1 km so the
// exact position is never stored.
func CoarsenLocation(lat, lng float64) (float64, float64, error) {
	if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, errLocation
	}
	return math.Round(lat*locationPrecision) / locationPrecision, math.Round(lng*locationPrecision) / locationPrecision, nil
}

// ApproximateDistance rounds a distance for display: at least 1 km, then
// whole kilometres, then 5 km steps beyond 20 km.
func ApproximateDistance(km float64) int {
	if km < 1 {
		return 1
	}
	if km < 20 {
		return int(math.Round(km))
	}
	return int(math.Round(km/5) * 5)
}
