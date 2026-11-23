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
	ErrInvalidFirstName    = errors.New("first name must be 1-40 letters")
	ErrInvalidBirthdate    = errors.New("birthdate must be a valid YYYY-MM-DD date")
	ErrUnderage            = errors.New("you must be at least 18 years old")
	ErrInvalidAge          = errors.New("age is out of range")
	ErrInvalidGender       = errors.New("gender must be woman, man or nonbinary")
	ErrBioTooLong          = errors.New("bio must be at most 500 characters")
	ErrCityTooLong         = errors.New("city must be at most 80 characters")
	ErrTooManyInterests    = errors.New("at most 10 interests")
	ErrUnknownInterest     = errors.New("unknown interest")
	ErrInvalidInterestedIn = errors.New("choose at least one valid gender to meet")
	ErrInvalidAgeRange     = errors.New("age range must be between 18 and 99")
	ErrInvalidDistance     = errors.New("distance must be between 1 and 300 km")
	ErrInvalidCoordinates  = errors.New("invalid coordinates")
)

// AgeOn returns the age in full years on the given day.
func AgeOn(birthdate, today time.Time) int {
	by, bm, bd := birthdate.Date()
	ty, tm, td := today.Date()
	age := ty - by
	if tm < bm || (tm == bm && td < bd) {
		age--
	}
	return age
}

// BirthdateBoundsForAges returns the [oldest, youngest] birthdates for people
// whose age is within [minAge, maxAge] on today.
func BirthdateBoundsForAges(minAge, maxAge int, today time.Time) (time.Time, time.Time) {
	y, m, d := today.Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	youngest := day.AddDate(-minAge, 0, 0)
	oldest := day.AddDate(-(maxAge + 1), 0, 1)
	return oldest, youngest
}

// SnapCoordinate rounds a coordinate to 2 decimals (~1.1 km) so exact
// positions are never stored.
func SnapCoordinate(v float64) float64 {
	return math.Round(v*100) / 100
}

// ApproximateDistance rounds a distance to whole kilometers, minimum 1 km, so
// distance cannot be used to pinpoint someone.
func ApproximateDistance(km float64) int {
	rounded := int(math.Round(km))
	if rounded < 1 {
		return 1
	}
	return rounded
}

func ValidateCoordinates(lat, lon float64) error {
	if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) {
		return ErrInvalidCoordinates
	}
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return ErrInvalidCoordinates
	}
	return nil
}

func cleanText(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
	return strings.TrimSpace(s)
}

func NormalizeFirstName(raw string) (string, error) {
	name := strings.Join(strings.Fields(cleanText(raw)), " ")
	n := utf8.RuneCountInString(name)
	if n < 1 || n > MaxNameRunes {
		return "", ErrInvalidFirstName
	}
	for _, r := range name {
		if !(unicode.IsLetter(r) || r == ' ' || r == '-' || r == '\'' || r == '’' || unicode.Is(unicode.Mn, r)) {
			return "", ErrInvalidFirstName
		}
	}
	return name, nil
}

func ParseBirthdate(raw string, today time.Time) (time.Time, error) {
	birthdate, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, ErrInvalidBirthdate
	}
	age := AgeOn(birthdate, today)
	if age < MinAge {
		return time.Time{}, ErrUnderage
	}
	if age > MaxAge+21 {
		return time.Time{}, ErrInvalidAge
	}
	return birthdate, nil
}

func ValidGender(g string) bool {
	for _, allowed := range Genders {
		if g == allowed {
			return true
		}
	}
	return false
}

func NormalizeBio(raw string) (string, error) {
	bio := cleanText(raw)
	if utf8.RuneCountInString(bio) > MaxBioRunes {
		return "", ErrBioTooLong
	}
	return bio, nil
}

func NormalizeCity(raw string) (string, error) {
	city := strings.Join(strings.Fields(cleanText(raw)), " ")
	if utf8.RuneCountInString(city) > MaxCityRunes {
		return "", ErrCityTooLong
	}
	return city, nil
}

func NormalizeInterests(slugs []string) ([]string, error) {
	seen := make(map[string]struct{}, len(slugs))
	out := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		s := strings.ToLower(strings.TrimSpace(slug))
		if s == "" {
			continue
		}
		if len(s) > 40 {
			return nil, ErrUnknownInterest
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	if len(out) > MaxInterests {
		return nil, ErrTooManyInterests
	}
	return out, nil
}

func NormalizePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	seen := map[string]struct{}{}
	genders := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		g = strings.TrimSpace(g)
		if !ValidGender(g) {
			return Preferences{}, ErrInvalidInterestedIn
		}
		if _, dup := seen[g]; dup {
			continue
		}
		seen[g] = struct{}{}
		genders = append(genders, g)
	}
	if len(genders) == 0 {
		return Preferences{}, ErrInvalidInterestedIn
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, ErrInvalidAgeRange
	}
	if req.MaxDistanceKm < MinDistanceKm || req.MaxDistanceKm > MaxDistanceKm {
		return Preferences{}, ErrInvalidDistance
	}
	return Preferences{InterestedIn: genders, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}

// MissingFields lists what prevents a profile from being shown in discovery.
func MissingFields(p StoredProfile, prefs Preferences, photoCount int) []string {
	missing := make([]string, 0, 6)
	if p.FirstName == nil || *p.FirstName == "" {
		missing = append(missing, "firstName")
	}
	if p.Birthdate == nil {
		missing = append(missing, "birthdate")
	}
	if p.Gender == nil || *p.Gender == "" {
		missing = append(missing, "gender")
	}
	if len(prefs.InterestedIn) == 0 {
		missing = append(missing, "interestedIn")
	}
	if photoCount == 0 {
		missing = append(missing, "photo")
	}
	if p.Latitude == nil || p.Longitude == nil {
		missing = append(missing, "location")
	}
	return missing
}
