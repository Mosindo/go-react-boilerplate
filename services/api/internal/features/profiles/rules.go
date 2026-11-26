package profiles

import (
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	platformerrors "example.com/api/internal/platform/errors"
)

// AgeOn returns the age in full years at the given date.
func AgeOn(birthdate, today time.Time) int {
	age := today.Year() - birthdate.Year()
	if today.Month() < birthdate.Month() || (today.Month() == birthdate.Month() && today.Day() < birthdate.Day()) {
		age--
	}
	return age
}

func today() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// ParseBirthdate validates a YYYY-MM-DD birthdate against the product age rules (18-99).
func ParseBirthdate(raw string) (time.Time, error) {
	t, err := time.Parse(birthdateLayout, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, platformerrors.Validation("invalid_birthdate", "Date de naissance invalide (format AAAA-MM-JJ).")
	}
	age := AgeOn(t, today())
	if age < MinAge {
		return time.Time{}, platformerrors.Validation("underage", "Vous devez avoir au moins 18 ans pour utiliser Lueur.")
	}
	if age > MaxAge {
		return time.Time{}, platformerrors.Validation("invalid_birthdate", "Date de naissance invalide.")
	}
	return t, nil
}

// NormalizeFirstName trims and validates a first name: letters, spaces, hyphens, apostrophes.
func NormalizeFirstName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	n := utf8.RuneCountInString(name)
	if n < 1 || n > maxFirstNameLen {
		return "", platformerrors.Validation("invalid_first_name", "Le prénom doit contenir entre 1 et 40 caractères.")
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && r != ' ' && r != '-' && r != '\'' && r != '’' {
			return "", platformerrors.Validation("invalid_first_name", "Le prénom ne peut contenir que des lettres, espaces, tirets et apostrophes.")
		}
	}
	return name, nil
}

// NormalizeText trims, strips control characters and enforces a maximum rune length.
func NormalizeText(raw string, max int, code, message string) (string, error) {
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.TrimSpace(raw))
	if utf8.RuneCountInString(cleaned) > max {
		return "", platformerrors.Validation(code, message)
	}
	return cleaned, nil
}

func ValidGender(g string) bool {
	_, ok := validGenders[g]
	return ok
}

func ValidatePreferences(req UpdatePreferencesRequest) (Preferences, error) {
	seen := map[string]struct{}{}
	var genders []string
	for _, g := range req.InterestedIn {
		if !ValidGender(g) {
			return Preferences{}, platformerrors.Validation("invalid_interested_in", "Préférence de genre invalide.")
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			genders = append(genders, g)
		}
	}
	if len(genders) == 0 {
		return Preferences{}, platformerrors.Validation("invalid_interested_in", "Choisissez au moins un genre recherché.")
	}
	if req.MinAge < MinAge || req.MaxAge > MaxAge || req.MinAge > req.MaxAge {
		return Preferences{}, platformerrors.Validation("invalid_age_range", "La tranche d'âge doit être comprise entre 18 et 99 ans.")
	}
	if req.MaxDistanceKm < 1 || req.MaxDistanceKm > MaxDistanceKm {
		return Preferences{}, platformerrors.Validation("invalid_distance", "La distance maximale doit être comprise entre 1 et 500 km.")
	}
	return Preferences{InterestedIn: genders, MinAge: req.MinAge, MaxAge: req.MaxAge, MaxDistanceKm: req.MaxDistanceKm}, nil
}

// CoarsenCoordinate rounds to two decimals (~1.1 km of latitude) before storage,
// so the database never holds a precise position.
func CoarsenCoordinate(v float64) float64 {
	return math.Round(v*100) / 100
}

func HaversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusKm * math.Asin(math.Sqrt(a))
}

// ApproxDistanceKm turns a raw distance into what is displayed: whole kilometres, never below 1,
// and in 5 km steps beyond 10 km, which limits trilateration attempts.
func ApproxDistanceKm(d float64) int {
	if d <= 1 {
		return 1
	}
	if d <= 10 {
		return int(math.Round(d))
	}
	return int(math.Round(d/5) * 5)
}
