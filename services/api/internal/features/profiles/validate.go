package profiles

import (
	"errors"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ValidationError is a user-fixable input problem (HTTP 400).
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func invalid(msg string) error { return &ValidationError{Msg: msg} }

// IsValidation reports whether err is a ValidationError.
func IsValidation(err error) bool {
	var v *ValidationError
	return errors.As(err, &v)
}

// cleanText strips control and invisible formatting characters (keeping "\n" when allowNewlines),
// normalises line endings and trims the result.
func cleanText(s string, allowNewlines bool) string {
	s = strings.ToValidUTF8(s, "")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '\n' && allowNewlines:
			b.WriteRune(r)
		case unicode.IsControl(r), unicode.Is(unicode.Cf, r), r == ' ', r == ' ':
			// dropped
		default:
			b.WriteRune(r)
		}
	}
	return strings.TrimSpace(b.String())
}

// NormalizeName trims, strips control characters and collapses runs of whitespace.
func NormalizeName(s string) string {
	return strings.Join(strings.Fields(cleanText(s, false)), " ")
}

// ValidateFirstName returns the normalised first name (1-40 letters, spaces, hyphens, apostrophes).
func ValidateFirstName(raw string) (string, error) {
	name := NormalizeName(raw)
	n := utf8.RuneCountInString(name)
	if n < 1 || n > MaxFirstName {
		return "", invalid("firstName must be 1 to 40 characters")
	}
	letters := 0
	for _, r := range name {
		switch {
		case unicode.IsLetter(r):
			letters++
		case unicode.IsMark(r), r == ' ', r == '-', r == '\'', r == '’':
		default:
			return "", invalid("firstName may only contain letters, spaces, hyphens and apostrophes")
		}
	}
	if letters == 0 {
		return "", invalid("firstName must contain a letter")
	}
	return name, nil
}

// ValidateBio returns the cleaned bio (<= 500 characters, newlines kept).
func ValidateBio(raw string) (string, error) {
	bio := cleanText(raw, true)
	if utf8.RuneCountInString(bio) > MaxBio {
		return "", invalid("bio must be at most 500 characters")
	}
	return bio, nil
}

// ValidateCity returns the cleaned city (<= 80 characters).
func ValidateCity(raw string) (string, error) {
	city := NormalizeName(raw)
	if utf8.RuneCountInString(city) > MaxCity {
		return "", invalid("city must be at most 80 characters")
	}
	return city, nil
}

// NormalizeInterests lower-cases, trims and deduplicates slugs (order preserved) and enforces the cap.
func NormalizeInterests(raw []string) ([]string, error) {
	if len(raw) > maxRawInterests {
		return nil, invalid("too many interests")
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		slug := strings.ToLower(strings.TrimSpace(s))
		if slug == "" || len(slug) > 40 {
			return nil, invalid("invalid interest")
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		out = append(out, slug)
	}
	if len(out) > MaxInterests {
		return nil, invalid("at most 10 interests are allowed")
	}
	return out, nil
}

// ValidateProfile validates and normalises a profile write.
func ValidateProfile(req ProfileRequest) (ProfileInput, error) {
	name, err := ValidateFirstName(req.FirstName)
	if err != nil {
		return ProfileInput{}, err
	}
	g := Gender(strings.TrimSpace(req.Gender))
	if !g.valid() {
		return ProfileInput{}, invalid("gender must be one of woman, man, non_binary")
	}
	bio, err := ValidateBio(req.Bio)
	if err != nil {
		return ProfileInput{}, err
	}
	city, err := ValidateCity(req.City)
	if err != nil {
		return ProfileInput{}, err
	}
	interests, err := NormalizeInterests(req.Interests)
	if err != nil {
		return ProfileInput{}, err
	}
	return ProfileInput{
		FirstName: name, Gender: g, Bio: bio, City: city, Interests: interests,
		ShowDistance: req.ShowDistance, ShowAge: req.ShowAge, Discoverable: req.Discoverable,
	}, nil
}

// RoundCoordinate rounds to 2 decimals (~1 km) and normalises negative zero.
func RoundCoordinate(v float64) float64 {
	r := math.Round(v*locationDecimals) / locationDecimals
	if r == 0 {
		return 0
	}
	return r
}

// ValidateLocation checks ranges and returns coordinates rounded to 2 decimals.
func ValidateLocation(req LocationRequest) (lat, lng float64, err error) {
	if req.Latitude == nil || req.Longitude == nil {
		return 0, 0, invalid("latitude and longitude are required")
	}
	la, lo := *req.Latitude, *req.Longitude
	if math.IsNaN(la) || math.IsInf(la, 0) || la < -90 || la > 90 {
		return 0, 0, invalid("latitude must be between -90 and 90")
	}
	if math.IsNaN(lo) || math.IsInf(lo, 0) || lo < -180 || lo > 180 {
		return 0, 0, invalid("longitude must be between -180 and 180")
	}
	return RoundCoordinate(la), RoundCoordinate(lo), nil
}

// ValidatePreferences validates a preferences write.
func ValidatePreferences(req PreferencesRequest) (Preferences, error) {
	if len(req.InterestedIn) < 1 || len(req.InterestedIn) > len(AllGenders)*4 {
		return Preferences{}, invalid("interestedIn must contain at least one gender")
	}
	seen := map[Gender]bool{}
	var genders []Gender
	for _, s := range req.InterestedIn {
		g := Gender(strings.TrimSpace(s))
		if !g.valid() {
			return Preferences{}, invalid("interestedIn contains an invalid gender")
		}
		if !seen[g] {
			seen[g] = true
			genders = append(genders, g)
		}
	}
	if req.AgeMin == nil || req.AgeMax == nil || req.MaxDistanceKm == nil {
		return Preferences{}, invalid("ageMin, ageMax and maxDistanceKm are required")
	}
	if *req.AgeMin < MinPrefAge || *req.AgeMin > MaxPrefAge {
		return Preferences{}, invalid("ageMin must be between 18 and 99")
	}
	if *req.AgeMax > MaxPrefAge || *req.AgeMax < *req.AgeMin {
		return Preferences{}, invalid("ageMax must be between ageMin and 99")
	}
	if *req.MaxDistanceKm < MinPrefDistance || *req.MaxDistanceKm > MaxPrefDistance {
		return Preferences{}, invalid("maxDistanceKm must be between 1 and 500")
	}
	return Preferences{InterestedIn: genders, AgeMin: *req.AgeMin, AgeMax: *req.AgeMax, MaxDistanceKm: *req.MaxDistanceKm}, nil
}
