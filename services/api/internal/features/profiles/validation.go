package profiles

import (
	"errors"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalidFirstName    = errors.New("first name must be 1-40 letters")
	ErrInvalidBirthdate    = errors.New("birthdate must be a valid date (YYYY-MM-DD)")
	ErrTooYoung            = errors.New("you must be at least 18 years old")
	ErrInvalidAge          = errors.New("birthdate is not plausible")
	ErrInvalidGender       = errors.New("invalid gender")
	ErrBioTooLong          = errors.New("bio must be at most 500 characters")
	ErrCityTooLong         = errors.New("city must be at most 80 characters")
	ErrJobTooLong          = errors.New("job title must be at most 80 characters")
	ErrInvalidHeight       = errors.New("height must be between 120 and 230 cm")
	ErrInvalidGoal         = errors.New("invalid relationship goal")
	ErrInvalidInterestedIn = errors.New("choose at least one valid gender to meet")
	ErrInvalidAgeRange     = errors.New("age range must be within 18-99 and min <= max")
	ErrInvalidDistance     = errors.New("distance must be between 1 and 500 km")
	ErrTooManyInterests    = errors.New("choose at most 10 interests")
	ErrUnknownInterest     = errors.New("unknown interest")
	ErrInvalidLocation     = errors.New("invalid coordinates")
)

// Age returns the age in full years at now.
func Age(birthdate, now time.Time) int {
	years := now.Year() - birthdate.Year()
	if now.Month() < birthdate.Month() || (now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
		years--
	}
	return years
}

// sanitizeText trims, collapses internal whitespace runs on a single line and
// strips control characters.
func sanitizeLine(value string) string {
	fields := strings.FieldsFunc(value, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) })
	return strings.Join(fields, " ")
}

// sanitizeMultiline keeps newlines (max two in a row) but strips other control characters.
func sanitizeMultiline(value string) string {
	var b strings.Builder
	newlines := 0
	for _, r := range strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n")) {
		if r == '\n' {
			newlines++
			if newlines <= 2 {
				b.WriteRune(r)
			}
			continue
		}
		newlines = 0
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func validFirstName(name string) bool {
	count := utf8.RuneCountInString(name)
	if count < 1 || count > 40 {
		return false
	}
	letters := 0
	for _, r := range name {
		switch {
		case unicode.IsLetter(r):
			letters++
		case r == ' ' || r == '-' || r == '\'' || r == '’' || r == '.':
		default:
			return false
		}
	}
	return letters > 0
}

func contains(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

// ValidateProfileInput normalizes and validates a profile payload.
func ValidateProfileInput(req UpsertProfileRequest, now time.Time) (ProfileInput, error) {
	firstName := sanitizeLine(req.FirstName)
	if !validFirstName(firstName) {
		return ProfileInput{}, ErrInvalidFirstName
	}

	birthdate, err := time.Parse("2006-01-02", strings.TrimSpace(req.Birthdate))
	if err != nil {
		return ProfileInput{}, ErrInvalidBirthdate
	}
	age := Age(birthdate, now)
	if age < MinAge {
		return ProfileInput{}, ErrTooYoung
	}
	if age > MaxAge+20 {
		return ProfileInput{}, ErrInvalidAge
	}

	gender := strings.ToLower(strings.TrimSpace(req.Gender))
	if !contains(Genders, gender) {
		return ProfileInput{}, ErrInvalidGender
	}

	bio := sanitizeMultiline(req.Bio)
	if utf8.RuneCountInString(bio) > 500 {
		return ProfileInput{}, ErrBioTooLong
	}
	city := sanitizeLine(req.City)
	if utf8.RuneCountInString(city) > 80 {
		return ProfileInput{}, ErrCityTooLong
	}
	job := sanitizeLine(req.JobTitle)
	if utf8.RuneCountInString(job) > 80 {
		return ProfileInput{}, ErrJobTooLong
	}
	if req.HeightCm != nil && (*req.HeightCm < 120 || *req.HeightCm > 230) {
		return ProfileInput{}, ErrInvalidHeight
	}
	var goal *string
	if req.RelationshipGoal != nil && strings.TrimSpace(*req.RelationshipGoal) != "" {
		g := strings.TrimSpace(*req.RelationshipGoal)
		if !contains(RelationshipGoals, g) {
			return ProfileInput{}, ErrInvalidGoal
		}
		goal = &g
	}

	return ProfileInput{
		FirstName:        firstName,
		Birthdate:        birthdate,
		Gender:           gender,
		Bio:              bio,
		City:             city,
		JobTitle:         job,
		HeightCm:         req.HeightCm,
		RelationshipGoal: goal,
	}, nil
}

// ValidatePreferences normalizes and validates discovery preferences.
func ValidatePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	seen := map[string]struct{}{}
	interested := make([]string, 0, len(req.InterestedIn))
	for _, raw := range req.InterestedIn {
		g := strings.ToLower(strings.TrimSpace(raw))
		if !contains(Genders, g) {
			return Preferences{}, ErrInvalidInterestedIn
		}
		if _, dup := seen[g]; dup {
			continue
		}
		seen[g] = struct{}{}
		interested = append(interested, g)
	}
	if len(interested) == 0 {
		return Preferences{}, ErrInvalidInterestedIn
	}
	sort.Strings(interested)
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, ErrInvalidAgeRange
	}
	if req.MaxDistanceKm < MinDistanceKm || req.MaxDistanceKm > MaxDistanceKm {
		return Preferences{}, ErrInvalidDistance
	}
	return Preferences{
		InterestedIn:  interested,
		MinAge:        req.MinAge,
		MaxAge:        req.MaxAge,
		MaxDistanceKm: req.MaxDistanceKm,
	}, nil
}

// NormalizeInterestIDs deduplicates ids and enforces the maximum.
func NormalizeInterestIDs(ids []int) ([]int, error) {
	seen := map[int]struct{}{}
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if id <= 0 || id > 32767 {
			return nil, ErrUnknownInterest
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) > MaxInterests {
		return nil, ErrTooManyInterests
	}
	sort.Ints(out)
	return out, nil
}

// Missing lists what still prevents a profile from appearing in discovery.
func Missing(profile *Profile, photoCount int) []string {
	missing := []string{}
	if profile == nil {
		missing = append(missing, "profile")
	}
	if photoCount == 0 {
		missing = append(missing, "photo")
	}
	return missing
}
