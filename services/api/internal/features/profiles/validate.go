package profiles

import (
	"math"
	"strings"
	"time"
	"unicode"

	platformerrors "example.com/api/internal/platform/errors"
)

// cleanText trims, collapses runs of whitespace and drops control characters.
func cleanText(s string, multiline bool) string {
	var b strings.Builder
	space := false
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r == '\n' && multiline:
			b.WriteRune(r)
			space = false
		case unicode.IsSpace(r):
			if !space {
				b.WriteRune(' ')
				space = true
			}
		case unicode.IsControl(r):
		default:
			b.WriteRune(r)
			space = false
		}
	}
	return strings.TrimSpace(b.String())
}

func runeLen(s string) int { return len([]rune(s)) }

func validName(s string) bool {
	for _, r := range s {
		if !(unicode.IsLetter(r) || unicode.IsMark(r) || r == ' ' || r == '-' || r == '\'' || r == '’') {
			return false
		}
	}
	return true
}

// AgeOn returns the full years between birthdate and now.
func AgeOn(birthdate, now time.Time) int {
	age := now.Year() - birthdate.Year()
	if now.Month() < birthdate.Month() || (now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
		age--
	}
	return age
}

type validProfile struct {
	FirstName  string
	Birthdate  time.Time
	Gender     string
	Bio        string
	Occupation string
	LookingFor string
	City       string
	Interests  []string
}

func validateProfile(req UpdateProfileRequest, now time.Time) (validProfile, error) {
	var v validProfile
	v.FirstName = cleanText(req.FirstName, false)
	if n := runeLen(v.FirstName); n < 1 || n > 40 || !validName(v.FirstName) {
		return v, platformerrors.BadRequest("invalid_first_name", "first name must be 1 to 40 letters")
	}
	bd, err := time.Parse("2006-01-02", strings.TrimSpace(req.Birthdate))
	if err != nil {
		return v, platformerrors.BadRequest("invalid_birthdate", "birthdate must be formatted YYYY-MM-DD")
	}
	age := AgeOn(bd, now)
	if age < MinAge {
		return v, platformerrors.Forbidden("underage", "you must be at least 18 years old")
	}
	if age > MaxAge {
		return v, platformerrors.BadRequest("invalid_birthdate", "invalid birthdate")
	}
	v.Birthdate = bd
	if !validGenders[req.Gender] {
		return v, platformerrors.BadRequest("invalid_gender", "invalid gender")
	}
	v.Gender = req.Gender
	v.Bio = cleanText(req.Bio, true)
	if runeLen(v.Bio) > 500 {
		return v, platformerrors.BadRequest("invalid_bio", "bio must be at most 500 characters")
	}
	v.Occupation = cleanText(req.Occupation, false)
	if runeLen(v.Occupation) > 80 {
		return v, platformerrors.BadRequest("invalid_occupation", "occupation must be at most 80 characters")
	}
	v.LookingFor = req.LookingFor
	if v.LookingFor == "" {
		v.LookingFor = "open"
	}
	if !validLookingFor[v.LookingFor] {
		return v, platformerrors.BadRequest("invalid_looking_for", "invalid relationship goal")
	}
	v.City = cleanText(req.City, false)
	if runeLen(v.City) > 80 {
		return v, platformerrors.BadRequest("invalid_city", "city must be at most 80 characters")
	}
	seen := map[string]bool{}
	for _, slug := range req.Interests {
		slug = strings.TrimSpace(slug)
		if slug != "" && !seen[slug] {
			seen[slug] = true
			v.Interests = append(v.Interests, slug)
		}
	}
	if len(v.Interests) > MaxInterests {
		return v, platformerrors.BadRequest("too_many_interests", "at most 10 interests")
	}
	return v, nil
}

func validatePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	p := Preferences{MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm, IsVisible: req.IsVisible, ShowDistance: req.ShowDistance}
	seen := map[string]bool{}
	for _, g := range req.InterestedIn {
		if !validGenders[g] {
			return p, platformerrors.BadRequest("invalid_gender", "invalid gender in interestedIn")
		}
		if !seen[g] {
			seen[g] = true
			p.InterestedIn = append(p.InterestedIn, g)
		}
	}
	if len(p.InterestedIn) == 0 {
		return p, platformerrors.BadRequest("invalid_interested_in", "choose at least one gender")
	}
	if p.MinAge < MinAge || p.MaxAge > MaxAge || p.MinAge > p.MaxAge {
		return p, platformerrors.BadRequest("invalid_age_range", "age range must be within 18-99 and min <= max")
	}
	if p.MaxDistanceKm != nil && (*p.MaxDistanceKm < 1 || *p.MaxDistanceKm > 20000) {
		return p, platformerrors.BadRequest("invalid_distance", "distance must be between 1 and 20000 km")
	}
	return p, nil
}

// snapCoordinate rounds to 0.01 degree (~1 km) so exact positions are never stored.
func snapCoordinate(v float64) float64 { return math.Round(v*100) / 100 }

func validateLocation(req UpdateLocationRequest) (lat, lng float64, city string, err error) {
	if req.Latitude == nil || req.Longitude == nil || math.IsNaN(*req.Latitude) || math.IsNaN(*req.Longitude) ||
		*req.Latitude < -90 || *req.Latitude > 90 || *req.Longitude < -180 || *req.Longitude > 180 {
		return 0, 0, "", platformerrors.BadRequest("invalid_location", "invalid coordinates")
	}
	city = cleanText(req.City, false)
	if runeLen(city) > 80 {
		return 0, 0, "", platformerrors.BadRequest("invalid_city", "city must be at most 80 characters")
	}
	return snapCoordinate(*req.Latitude), snapCoordinate(*req.Longitude), city, nil
}

// DisplayDistance converts a raw distance to a coarse bucket (5 km steps, minimum 5)
// so repeated queries cannot be used to triangulate someone.
func DisplayDistance(km float64) int {
	if km <= 5 {
		return 5
	}
	return int(math.Ceil(km/5)) * 5
}
