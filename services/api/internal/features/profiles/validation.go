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

var (
	ErrInvalidFirstName    = errors.New("first name must be 1 to 40 letters")
	ErrInvalidBirthdate    = errors.New("birthdate must be a valid YYYY-MM-DD date")
	ErrTooYoung            = errors.New("you must be at least 18 years old")
	ErrInvalidAge          = errors.New("birthdate is not plausible")
	ErrInvalidGender       = errors.New("gender must be woman, man or nonbinary")
	ErrBioTooLong          = errors.New("bio must be at most 500 characters")
	ErrJobTitleTooLong     = errors.New("job title must be at most 60 characters")
	ErrInvalidHeight       = errors.New("height must be between 120 and 230 cm")
	ErrInvalidGoal         = errors.New("relationship goal is not supported")
	ErrTooManyInterests    = errors.New("choose at most 10 interests")
	ErrUnknownInterest     = errors.New("unknown interest")
	ErrInvalidLocation     = errors.New("latitude/longitude are out of range")
	ErrCityTooLong         = errors.New("city must be at most 80 characters")
	ErrInvalidInterestedIn = errors.New("choose at least one gender you are interested in")
	ErrInvalidAgeRange     = errors.New("age range must be within 18-99 and min <= max")
	ErrInvalidDistance     = errors.New("distance must be between 1 and 500 km")
)

// ValidationError marks errors that are safe to show to the client.
func IsValidationError(err error) bool {
	for _, known := range []error{
		ErrInvalidFirstName, ErrInvalidBirthdate, ErrTooYoung, ErrInvalidAge, ErrInvalidGender,
		ErrBioTooLong, ErrJobTitleTooLong, ErrInvalidHeight, ErrInvalidGoal, ErrTooManyInterests,
		ErrUnknownInterest, ErrInvalidLocation, ErrCityTooLong, ErrInvalidInterestedIn,
		ErrInvalidAgeRange, ErrInvalidDistance,
	} {
		if errors.Is(err, known) {
			return true
		}
	}
	return false
}

// AgeOn returns the age in full years at the given instant.
func AgeOn(birthdate, now time.Time) int {
	years := now.Year() - birthdate.Year()
	if now.Month() < birthdate.Month() || (now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
		years--
	}
	return years
}

func normalizeText(v string) string {
	// Collapse internal whitespace runs and strip control characters.
	var b strings.Builder
	space := false
	for _, r := range strings.TrimSpace(v) {
		if unicode.IsControl(r) && r != '\n' {
			continue
		}
		if r == ' ' || r == '\t' {
			if space {
				continue
			}
			space = true
			b.WriteRune(' ')
			continue
		}
		space = false
		b.WriteRune(r)
	}
	return b.String()
}

func validateFirstName(v string) (string, error) {
	name := normalizeText(strings.ReplaceAll(v, "\n", " "))
	n := utf8.RuneCountInString(name)
	if n == 0 || n > MaxFirstNameRunes {
		return "", ErrInvalidFirstName
	}
	for _, r := range name {
		if !(unicode.IsLetter(r) || r == ' ' || r == '-' || r == '\'' || r == '’' || unicode.Is(unicode.Mn, r)) {
			return "", ErrInvalidFirstName
		}
	}
	return name, nil
}

func validateBirthdate(v string, now time.Time) (time.Time, error) {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(v))
	if err != nil {
		return time.Time{}, ErrInvalidBirthdate
	}
	if date.After(now) {
		return time.Time{}, ErrInvalidBirthdate
	}
	age := AgeOn(date, now)
	if age < MinAge {
		return time.Time{}, ErrTooYoung
	}
	if age > 100 {
		return time.Time{}, ErrInvalidAge
	}
	return date, nil
}

func validateGender(v string) (string, error) {
	g := strings.ToLower(strings.TrimSpace(v))
	if !slices.Contains(Genders, g) {
		return "", ErrInvalidGender
	}
	return g, nil
}

func validateInterestedIn(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	for _, v := range values {
		g, err := validateGender(v)
		if err != nil {
			return nil, ErrInvalidInterestedIn
		}
		if !slices.Contains(out, g) {
			out = append(out, g)
		}
	}
	if len(out) == 0 {
		return nil, ErrInvalidInterestedIn
	}
	slices.Sort(out)
	return out, nil
}

func validatePatch(req UpdateProfileRequest, now time.Time) (profilePatch, error) {
	var p profilePatch
	if req.FirstName != nil {
		name, err := validateFirstName(*req.FirstName)
		if err != nil {
			return p, err
		}
		p.FirstName = &name
	}
	if req.Birthdate != nil {
		date, err := validateBirthdate(*req.Birthdate, now)
		if err != nil {
			return p, err
		}
		p.Birthdate = &date
	}
	if req.Gender != nil {
		g, err := validateGender(*req.Gender)
		if err != nil {
			return p, err
		}
		p.Gender = &g
	}
	if req.Bio != nil {
		bio := strings.TrimSpace(*req.Bio)
		if utf8.RuneCountInString(bio) > MaxBioRunes {
			return p, ErrBioTooLong
		}
		p.Bio = &bio
	}
	if req.JobTitle != nil {
		job := normalizeText(strings.ReplaceAll(*req.JobTitle, "\n", " "))
		if utf8.RuneCountInString(job) > MaxJobTitleRunes {
			return p, ErrJobTitleTooLong
		}
		p.JobTitle = &job
	}
	if req.HeightCm != nil {
		if *req.HeightCm == 0 {
			p.ClearHeight = true
		} else if *req.HeightCm < MinHeightCm || *req.HeightCm > MaxHeightCm {
			return p, ErrInvalidHeight
		} else {
			h := *req.HeightCm
			p.HeightCm = &h
		}
	}
	if req.RelationshipGoal != nil {
		goal := strings.ToLower(strings.TrimSpace(*req.RelationshipGoal))
		if goal == "" {
			p.ClearRelationGoal = true
		} else if !slices.Contains(RelationshipGoals, goal) {
			return p, ErrInvalidGoal
		} else {
			p.RelationshipGoal = &goal
		}
	}
	if req.Interests != nil {
		slugs := make([]string, 0, len(*req.Interests))
		for _, raw := range *req.Interests {
			slug := strings.ToLower(strings.TrimSpace(raw))
			if slug == "" || len(slug) > 40 {
				return p, ErrUnknownInterest
			}
			if !slices.Contains(slugs, slug) {
				slugs = append(slugs, slug)
			}
		}
		if len(slugs) > MaxInterests {
			return p, ErrTooManyInterests
		}
		p.InterestSlugs = &slugs
	}
	p.IsPaused = req.IsPaused
	p.ShowDistance = req.ShowDistance
	return p, nil
}

// RoundCoordinate keeps two decimals (~1.1 km) so the exact position of a
// member is never stored.
func RoundCoordinate(v float64) float64 {
	return math.Round(v*100) / 100
}

func validateLocation(lat, lon float64, city string) (float64, float64, string, error) {
	if math.IsNaN(lat) || math.IsNaN(lon) || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return 0, 0, "", ErrInvalidLocation
	}
	c := normalizeText(strings.ReplaceAll(city, "\n", " "))
	if utf8.RuneCountInString(c) > MaxCityRunes {
		return 0, 0, "", ErrCityTooLong
	}
	return RoundCoordinate(lat), RoundCoordinate(lon), c, nil
}

func applyPreferences(current Preferences, req UpdatePreferencesRequest) (Preferences, error) {
	next := current
	if req.InterestedIn != nil {
		values, err := validateInterestedIn(*req.InterestedIn)
		if err != nil {
			return current, err
		}
		next.InterestedIn = values
	}
	if req.AgeMin != nil {
		next.AgeMin = *req.AgeMin
	}
	if req.AgeMax != nil {
		next.AgeMax = *req.AgeMax
	}
	if next.AgeMin < MinAge || next.AgeMax > MaxAge || next.AgeMin > next.AgeMax {
		return current, ErrInvalidAgeRange
	}
	if req.MaxDistanceKm != nil {
		next.MaxDistanceKm = *req.MaxDistanceKm
	}
	if next.MaxDistanceKm < 1 || next.MaxDistanceKm > MaxDistanceKm {
		return current, ErrInvalidDistance
	}
	return next, nil
}

// ApproximateDistance turns a raw distance into a coarse, display-safe value:
// rounded up to the kilometre, never below 1, and bucketed by 5 km beyond 10.
func ApproximateDistance(km float64) int {
	if km <= 1 {
		return 1
	}
	rounded := int(math.Ceil(km))
	if rounded > 10 {
		rounded = int(math.Ceil(km/5)) * 5
	}
	return rounded
}

// missingFields lists what still blocks discovery, for onboarding UX.
func missingFields(p storedProfile, photoCount int) []string {
	missing := make([]string, 0)
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
	if !p.HasLocation {
		missing = append(missing, "location")
	}
	return missing
}
