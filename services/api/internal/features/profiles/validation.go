package profiles

import (
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
)

const (
	MinAge          = 18
	MaxAge          = 99
	maxInterests    = 10
	maxBioRunes     = 500
	maxNameRunes    = 40
	maxCityRunes    = 80
	locationGridDeg = 0.02 // ~2 km: coarse enough that exact positions are never stored
)

// AgeOn returns the full years between birth and now.
func AgeOn(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

// RoundCoordinate snaps a coordinate to the storage grid.
func RoundCoordinate(v float64) float64 {
	r := math.Round(v/locationGridDeg) * locationGridDeg
	return math.Round(r*1e4) / 1e4
}

func validateText(s string, maxRunes int, allowNewline bool, allow func(rune) bool) bool {
	if utf8.RuneCountInString(s) > maxRunes || !utf8.ValidString(s) {
		return false
	}
	for _, r := range s {
		if allowNewline && (r == '\n' || r == '\r') {
			continue
		}
		if unicode.IsControl(r) || !allow(r) {
			return false
		}
	}
	return true
}

func nameRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsMark(r) || r == ' ' || r == '-' || r == '\'' || r == '’' || r == '.'
}

func cityRune(r rune) bool {
	return nameRune(r) || unicode.IsDigit(r) || r == ',' || r == '(' || r == ')'
}

func anyRune(rune) bool { return true }

// normalizeProfile validates input and returns the persisted form plus interest slugs.
func normalizeProfile(userID string, in ProfileInput, now time.Time) (StoredProfile, []string, error) {
	fields := map[string]string{}

	first := strings.TrimSpace(in.FirstName)
	if first == "" {
		fields["firstName"] = "required"
	} else if !validateText(first, maxNameRunes, false, nameRune) {
		fields["firstName"] = "invalid first name"
	}

	var birth time.Time
	if parsed, err := time.Parse("2006-01-02", in.BirthDate); err != nil {
		fields["birthDate"] = "must be a YYYY-MM-DD date"
	} else if age := AgeOn(parsed, now); age < MinAge {
		fields["birthDate"] = "you must be at least 18"
	} else if age > MaxAge {
		fields["birthDate"] = "invalid birth date"
	} else {
		birth = parsed
	}

	if _, ok := validGenders[in.Gender]; !ok {
		fields["gender"] = "must be man, woman or non_binary"
	}

	bio := strings.TrimSpace(in.Bio)
	if !validateText(bio, maxBioRunes, true, anyRune) {
		fields["bio"] = "must be at most 500 characters"
	}

	city := strings.TrimSpace(in.City)
	if city == "" {
		fields["city"] = "required"
	} else if !validateText(city, maxCityRunes, false, cityRune) {
		fields["city"] = "invalid city"
	}

	var lat, lng *float64
	switch {
	case in.ClearLocation:
		// Location is removed; coordinates in the same payload are ignored.
	case (in.Latitude == nil) != (in.Longitude == nil):
		fields["latitude"] = "latitude and longitude must be provided together"
	case in.Latitude != nil:
		la, lo := *in.Latitude, *in.Longitude
		if math.IsNaN(la) || math.IsNaN(lo) || la < -90 || la > 90 || lo < -180 || lo > 180 {
			fields["latitude"] = "coordinates out of range"
		} else {
			rl, ro := RoundCoordinate(la), RoundCoordinate(lo)
			lat, lng = &rl, &ro
		}
	}

	slugs := make([]string, 0, len(in.Interests))
	seen := map[string]struct{}{}
	for _, s := range in.Interests {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" || len(s) > 40 {
			fields["interests"] = "invalid interest"
			break
		}
		if _, dup := seen[s]; !dup {
			seen[s] = struct{}{}
			slugs = append(slugs, s)
		}
	}
	if len(slugs) > maxInterests {
		fields["interests"] = "at most 10 interests"
	}

	if len(fields) > 0 {
		return StoredProfile{}, nil, platformerrors.Validation(fields)
	}
	return StoredProfile{
		UserID: userID, FirstName: first, BirthDate: birth, Gender: in.Gender,
		Bio: bio, City: city, Latitude: lat, Longitude: lng,
	}, slugs, nil
}

func validatePreferences(p Preferences) (Preferences, error) {
	fields := map[string]string{}
	seen := map[string]struct{}{}
	genders := make([]string, 0, 3)
	for _, g := range p.InterestedIn {
		if _, ok := validGenders[g]; !ok {
			fields["interestedIn"] = "values must be man, woman or non_binary"
			break
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			genders = append(genders, g)
		}
	}
	if len(genders) == 0 && fields["interestedIn"] == "" {
		fields["interestedIn"] = "select at least one"
	}
	if p.MinAge < MinAge || p.MinAge > MaxAge {
		fields["minAge"] = "must be between 18 and 99"
	}
	if p.MaxAge < MinAge || p.MaxAge > MaxAge {
		fields["maxAge"] = "must be between 18 and 99"
	} else if p.MaxAge < p.MinAge {
		fields["maxAge"] = "must be greater than or equal to minAge"
	}
	if p.MaxDistanceKm < 1 || p.MaxDistanceKm > 500 {
		fields["maxDistanceKm"] = "must be between 1 and 500"
	}
	if len(fields) > 0 {
		return Preferences{}, platformerrors.Validation(fields)
	}
	p.InterestedIn = genders
	return p, nil
}
