package profiles

import (
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)

// AgeOn returns the full years between birth and now.
func AgeOn(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

// ApproxDistanceKm turns an exact distance into a coarse one: 1 km minimum,
// whole km up to 10 km, then steps of 5 km.
func ApproxDistanceKm(km float64) int {
	if km <= 1 {
		return 1
	}
	if km <= 10 {
		return int(math.Ceil(km))
	}
	return int(math.Ceil(km/5)) * 5
}

// roundCoordinate keeps roughly 1 km of precision, which is all the app needs.
func roundCoordinate(v float64) float64 {
	return math.Round(v*100) / 100
}

// cleanText trims, drops control characters (keeping newlines when allowed)
// and enforces a rune limit.
func cleanText(field, value string, max int, allowNewlines bool) (string, error) {
	value = strings.TrimSpace(value)
	var b strings.Builder
	for _, r := range value {
		if r == '\n' && allowNewlines {
			b.WriteRune(r)
			continue
		}
		if r == '\r' {
			continue
		}
		if unicode.IsControl(r) {
			return "", platformerrors.Invalid(field, "contains invalid characters")
		}
		b.WriteRune(r)
	}
	out := b.String()
	if utf8.RuneCountInString(out) > max {
		return "", platformerrors.Invalid(field, "is too long")
	}
	return out, nil
}

func validateProfile(req SaveProfileRequest, now time.Time) (ProfileInput, error) {
	name, err := cleanText("displayName", req.DisplayName, 50, false)
	if err != nil {
		return ProfileInput{}, err
	}
	if name == "" {
		return ProfileInput{}, platformerrors.Invalid("displayName", "is required")
	}

	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return ProfileInput{}, platformerrors.Invalid("birthDate", "must use the YYYY-MM-DD format")
	}
	if age := AgeOn(birth, now); age < MinAge {
		return ProfileInput{}, platformerrors.Invalid("birthDate", "you must be at least 18 years old")
	} else if age > MaxAge {
		return ProfileInput{}, platformerrors.Invalid("birthDate", "is not a valid birth date")
	}

	gender := strings.TrimSpace(req.Gender)
	if _, ok := validGenders[gender]; !ok {
		return ProfileInput{}, platformerrors.Invalid("gender", "must be woman, man or non_binary")
	}

	bio, err := cleanText("bio", req.Bio, 500, true)
	if err != nil {
		return ProfileInput{}, err
	}
	city, err := cleanText("city", req.City, 100, false)
	if err != nil {
		return ProfileInput{}, err
	}

	if (req.Latitude == nil) != (req.Longitude == nil) {
		return ProfileInput{}, platformerrors.Invalid("location", "latitude and longitude must be sent together")
	}
	var lat, lon *float64
	if req.Latitude != nil {
		la, lo := *req.Latitude, *req.Longitude
		if math.IsNaN(la) || math.IsNaN(lo) || la < -90 || la > 90 || lo < -180 || lo > 180 {
			return ProfileInput{}, platformerrors.Invalid("location", "is out of range")
		}
		la, lo = roundCoordinate(la), roundCoordinate(lo)
		lat, lon = &la, &lo
	}

	slugs, err := validateSlugs(req.Interests)
	if err != nil {
		return ProfileInput{}, err
	}

	return ProfileInput{
		DisplayName:    name,
		BirthDate:      birth,
		Gender:         gender,
		Bio:            bio,
		City:           city,
		Latitude:       lat,
		Longitude:      lon,
		InterestSlugs:  slugs,
		IsDiscoverable: req.IsDiscoverable,
		ShowDistance:   req.ShowDistance,
	}, nil
}

func validateSlugs(raw []string) ([]string, error) {
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, slug := range raw {
		slug = strings.ToLower(strings.TrimSpace(slug))
		if !slugPattern.MatchString(slug) {
			return nil, platformerrors.Invalid("interests", "contains an invalid interest")
		}
		if _, dup := seen[slug]; dup {
			continue
		}
		seen[slug] = struct{}{}
		out = append(out, slug)
	}
	if len(out) > MaxInterests {
		return nil, platformerrors.Invalid("interests", "at most 10 interests are allowed")
	}
	return out, nil
}

func validatePreferences(req SavePreferencesRequest) (Preferences, error) {
	seen := make(map[string]struct{}, 3)
	interested := make([]string, 0, 3)
	for _, g := range req.InterestedIn {
		g = strings.TrimSpace(g)
		if _, ok := validGenders[g]; !ok {
			return Preferences{}, platformerrors.Invalid("interestedIn", "contains an invalid gender")
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			interested = append(interested, g)
		}
	}
	if len(interested) == 0 {
		return Preferences{}, platformerrors.Invalid("interestedIn", "choose at least one option")
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, platformerrors.Invalid("minAge", "age range must be within 18-99 and min must not exceed max")
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > 1000 {
		return Preferences{}, platformerrors.Invalid("maxDistanceKm", "must be between 1 and 1000")
	}
	return Preferences{InterestedIn: interested, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}
