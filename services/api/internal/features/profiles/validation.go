package profiles

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalidFirstName  = errors.New("first name must be 1 to 40 characters and contain letters")
	ErrInvalidBirthdate  = errors.New("birthdate must use the YYYY-MM-DD format")
	ErrUnderage          = errors.New("you must be at least 18 years old")
	ErrInvalidAge        = errors.New("birthdate is not plausible")
	ErrInvalidGender     = errors.New("gender must be one of woman, man, nonbinary")
	ErrInvalidBio        = errors.New("bio must be 500 characters or fewer")
	ErrInvalidJobTitle   = errors.New("job title must be 60 characters or fewer")
	ErrInvalidCity       = errors.New("city must be 80 characters or fewer")
	ErrInvalidLookingFor = errors.New("lookingFor must be one of relationship, casual, friendship, unsure")
	ErrTooManyInterests  = errors.New("choose at most 10 interests")
	ErrInvalidInterest   = errors.New("unknown interest")
	ErrInvalidPrefs      = errors.New("invalid preferences")
	ErrInvalidLocation   = errors.New("invalid coordinates")
)

// AgeAt returns the age in full years at a given instant.
func AgeAt(birthdate, now time.Time) int {
	years := now.Year() - birthdate.Year()
	if now.Month() < birthdate.Month() || (now.Month() == birthdate.Month() && now.Day() < birthdate.Day()) {
		years--
	}
	return years
}

// ParseBirthdate validates a YYYY-MM-DD date and enforces the minimum age.
func ParseBirthdate(raw string, now time.Time) (time.Time, error) {
	d, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, ErrInvalidBirthdate
	}
	age := AgeAt(d, now)
	if age < MinAge {
		return time.Time{}, ErrUnderage
	}
	if age > MaxAge {
		return time.Time{}, ErrInvalidAge
	}
	return d, nil
}

// cleanText trims, collapses internal whitespace runs on single-line fields and strips control characters.
func cleanText(value string, multiline bool) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(value) {
		if r == '\n' && multiline {
			b.WriteRune(r)
			continue
		}
		if unicode.IsControl(r) {
			if !multiline {
				b.WriteRune(' ')
			}
			continue
		}
		b.WriteRune(r)
	}
	out := b.String()
	if !multiline {
		out = strings.Join(strings.Fields(out), " ")
	} else {
		for strings.Contains(out, "\n\n\n") {
			out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
		}
	}
	return out
}

type normalizedProfile struct {
	FirstName   string
	Birthdate   time.Time
	Gender      string
	Bio         string
	JobTitle    string
	LookingFor  string
	City        string
	InterestIDs []int
}

func normalizeProfile(req UpsertProfileRequest, now time.Time) (normalizedProfile, error) {
	out := normalizedProfile{
		FirstName:  cleanText(req.FirstName, false),
		Gender:     strings.ToLower(strings.TrimSpace(req.Gender)),
		Bio:        cleanText(req.Bio, true),
		JobTitle:   cleanText(req.JobTitle, false),
		LookingFor: strings.ToLower(strings.TrimSpace(req.LookingFor)),
		City:       cleanText(req.City, false),
	}
	if n := utf8.RuneCountInString(out.FirstName); n < 1 || n > 40 || !strings.ContainsFunc(out.FirstName, unicode.IsLetter) {
		return out, ErrInvalidFirstName
	}
	birthdate, err := ParseBirthdate(req.Birthdate, now)
	if err != nil {
		return out, err
	}
	out.Birthdate = birthdate
	if !slices.Contains(Genders, out.Gender) {
		return out, ErrInvalidGender
	}
	if utf8.RuneCountInString(out.Bio) > 500 {
		return out, ErrInvalidBio
	}
	if utf8.RuneCountInString(out.JobTitle) > 60 {
		return out, ErrInvalidJobTitle
	}
	if utf8.RuneCountInString(out.City) > 80 {
		return out, ErrInvalidCity
	}
	if out.LookingFor == "" {
		out.LookingFor = "unsure"
	}
	if !slices.Contains(LookingFors, out.LookingFor) {
		return out, ErrInvalidLookingFor
	}
	ids := slices.Clone(req.InterestIDs)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if len(ids) > MaxInterests {
		return out, ErrTooManyInterests
	}
	for _, id := range ids {
		if id <= 0 || id > 32767 {
			return out, ErrInvalidInterest
		}
	}
	out.InterestIDs = ids
	return out, nil
}

func normalizePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	genders := make([]string, 0, len(req.InterestedIn))
	for _, g := range req.InterestedIn {
		g = strings.ToLower(strings.TrimSpace(g))
		if !slices.Contains(Genders, g) {
			return Preferences{}, ErrInvalidPrefs
		}
		if !slices.Contains(genders, g) {
			genders = append(genders, g)
		}
	}
	if len(genders) == 0 {
		return Preferences{}, ErrInvalidPrefs
	}
	if req.AgeMin < MinAge || req.AgeMax > MaxAge || req.AgeMin > req.AgeMax {
		return Preferences{}, ErrInvalidPrefs
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > 500 {
		return Preferences{}, ErrInvalidPrefs
	}
	slices.Sort(genders)
	return Preferences{
		InterestedIn:   genders,
		AgeMin:         req.AgeMin,
		AgeMax:         req.AgeMax,
		MaxDistanceKm:  req.MaxDistanceKm,
		IsDiscoverable: *req.IsDiscoverable,
		ShowDistance:   *req.ShowDistance,
	}, nil
}
