package profiles

import (
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	perr "example.com/api/internal/platform/errors"
)

// AgeOn returns the age in full years on the given day.
func AgeOn(birth, now time.Time) int {
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	return age
}

// RoundCoordinate keeps ~1 km precision (0.01 degree). We never store anything
// more precise than what is needed to filter by distance.
func RoundCoordinate(v float64) float64 { return math.Round(v*100) / 100 }

// cleanText trims, collapses control characters and enforces a rune limit.
func cleanText(s string, max int) (string, bool) {
	s = strings.TrimSpace(s)
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' {
			return "", false
		}
	}
	if utf8.RuneCountInString(s) > max {
		return "", false
	}
	return s, true
}

type validProfile struct {
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	City         string
	Latitude     *float64
	Longitude    *float64
	ShowDistance bool
	Discoverable bool
	InterestIDs  []int
}

func validateProfile(req UpsertProfileRequest, now time.Time) (validProfile, error) {
	var v validProfile
	name, ok := cleanText(req.FirstName, MaxName)
	if !ok || name == "" || strings.Contains(name, "\n") {
		return v, perr.BadRequest("invalid_first_name", "Le prénom doit contenir entre 1 et 40 caractères.")
	}
	v.FirstName = name

	birth, err := time.Parse("2006-01-02", strings.TrimSpace(req.BirthDate))
	if err != nil {
		return v, perr.BadRequest("invalid_birth_date", "La date de naissance doit être au format AAAA-MM-JJ.")
	}
	age := AgeOn(birth, now)
	if age < MinAge {
		return v, perr.BadRequest("underage", "Tandem est réservé aux personnes de 18 ans et plus.")
	}
	if age > MaxAge {
		return v, perr.BadRequest("invalid_birth_date", "La date de naissance n'est pas valide.")
	}
	v.BirthDate = birth

	if !validGenders[req.Gender] {
		return v, perr.BadRequest("invalid_gender", "Genre invalide.")
	}
	v.Gender = req.Gender

	bio, ok := cleanText(req.Bio, MaxBio)
	if !ok {
		return v, perr.BadRequest("invalid_bio", "La bio ne doit pas dépasser 500 caractères.")
	}
	v.Bio = bio

	city, ok := cleanText(req.City, MaxCity)
	if !ok || strings.Contains(city, "\n") {
		return v, perr.BadRequest("invalid_city", "Le nom de la ville est invalide.")
	}
	v.City = city

	if (req.Latitude == nil) != (req.Longitude == nil) {
		return v, perr.BadRequest("invalid_location", "Latitude et longitude vont ensemble.")
	}
	if req.Latitude != nil {
		lat, lng := *req.Latitude, *req.Longitude
		if math.IsNaN(lat) || math.IsNaN(lng) || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
			return v, perr.BadRequest("invalid_location", "Position invalide.")
		}
		rl, rg := RoundCoordinate(lat), RoundCoordinate(lng)
		v.Latitude, v.Longitude = &rl, &rg
	}

	v.ShowDistance = req.ShowDistance == nil || *req.ShowDistance
	v.Discoverable = req.Discoverable == nil || *req.Discoverable

	if len(req.InterestIDs) > MaxInterests {
		return v, perr.BadRequest("too_many_interests", "10 centres d'intérêt maximum.")
	}
	seen := map[int]bool{}
	for _, id := range req.InterestIDs {
		if id <= 0 || seen[id] {
			return v, perr.BadRequest("invalid_interests", "Centres d'intérêt invalides.")
		}
		seen[id] = true
	}
	v.InterestIDs = req.InterestIDs
	return v, nil
}

func validatePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	var p Preferences
	if len(req.InterestedIn) == 0 || len(req.InterestedIn) > 3 {
		return p, perr.BadRequest("invalid_preferences", "Choisissez au moins un genre.")
	}
	seen := map[string]bool{}
	for _, g := range req.InterestedIn {
		if !validGenders[g] || seen[g] {
			return p, perr.BadRequest("invalid_preferences", "Genre recherché invalide.")
		}
		seen[g] = true
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return p, perr.BadRequest("invalid_preferences", "Tranche d'âge invalide (18-99).")
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > 500 {
		return p, perr.BadRequest("invalid_preferences", "Distance maximale invalide (1-500 km).")
	}
	return Preferences{InterestedIn: req.InterestedIn, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}
