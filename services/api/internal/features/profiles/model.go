package profiles

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrProfileRequired = errors.New("profile required")
	ErrInvalidInput    = errors.New("invalid input")
	ErrUnderage        = errors.New("must be at least 18 years old")
)

const (
	MinAge          = 18
	MaxAge          = 99
	MaxInterests    = 10
	maxFirstName    = 40
	maxBio          = 500
	maxCity         = 80
	birthDateLayout = "2006-01-02"
)

var validGenders = map[string]struct{}{"man": {}, "woman": {}, "nonbinary": {}}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type ProfileInput struct {
	FirstName string   `json:"firstName"`
	BirthDate string   `json:"birthDate"`
	Gender    string   `json:"gender"`
	Bio       string   `json:"bio"`
	City      string   `json:"city"`
	Latitude  float64  `json:"latitude"`
	Longitude float64  `json:"longitude"`
	Interests []string `json:"interests"`
}

// Profile est la vue privee (proprietaire) d'un profil.
type Profile struct {
	UserID    string     `json:"userId"`
	FirstName string     `json:"firstName"`
	BirthDate string     `json:"birthDate"`
	Age       int        `json:"age"`
	Gender    string     `json:"gender"`
	Bio       string     `json:"bio"`
	City      string     `json:"city"`
	Latitude  float64    `json:"latitude"`
	Longitude float64    `json:"longitude"`
	Interests []Interest `json:"interests"`
	PhotoIDs  []string   `json:"photoIds"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// PublicProfile est la vue d'un profil par un autre utilisateur: aucune coordonnee ni date de naissance.
type PublicProfile struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	PhotoIDs   []string   `json:"photoIds"`
}

type Settings struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
	Discoverable  bool     `json:"discoverable"`
	ShowDistance  bool     `json:"showDistance"`
}

func DefaultSettings() Settings {
	return Settings{InterestedIn: []string{"man", "woman", "nonbinary"}, MinAge: MinAge, MaxAge: MaxAge, MaxDistanceKm: 50, Discoverable: true, ShowDistance: true}
}

// AgeAt calcule l'age complet a la date `now`.
func AgeAt(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

// Normalize nettoie et valide une entree de profil. Retourne la date de naissance parsee.
func (in *ProfileInput) Normalize(now time.Time) (time.Time, error) {
	in.FirstName = strings.TrimSpace(in.FirstName)
	in.Bio = strings.TrimSpace(in.Bio)
	in.City = strings.TrimSpace(in.City)
	in.Gender = strings.ToLower(strings.TrimSpace(in.Gender))

	if n := utf8.RuneCountInString(in.FirstName); n < 1 || n > maxFirstName {
		return time.Time{}, ErrInvalidInput
	}
	if hasControl(in.FirstName) || hasControl(in.City) {
		return time.Time{}, ErrInvalidInput
	}
	if utf8.RuneCountInString(in.Bio) > maxBio || utf8.RuneCountInString(in.City) > maxCity {
		return time.Time{}, ErrInvalidInput
	}
	if _, ok := validGenders[in.Gender]; !ok {
		return time.Time{}, ErrInvalidInput
	}
	birth, err := time.Parse(birthDateLayout, in.BirthDate)
	if err != nil {
		return time.Time{}, ErrInvalidInput
	}
	age := AgeAt(birth, now)
	if age < MinAge {
		return time.Time{}, ErrUnderage
	}
	if age > MaxAge {
		return time.Time{}, ErrInvalidInput
	}
	if len(in.Interests) > MaxInterests {
		return time.Time{}, ErrInvalidInput
	}
	seen := map[string]struct{}{}
	clean := make([]string, 0, len(in.Interests))
	for _, s := range in.Interests {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" || len(s) > 40 {
			return time.Time{}, ErrInvalidInput
		}
		if _, dup := seen[s]; !dup {
			seen[s] = struct{}{}
			clean = append(clean, s)
		}
	}
	in.Interests = clean
	return birth, nil
}

func hasControl(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}

// Validate normalise et valide les reglages.
func (s *Settings) Validate() error {
	if len(s.InterestedIn) < 1 || len(s.InterestedIn) > 3 {
		return ErrInvalidInput
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, 3)
	for _, g := range s.InterestedIn {
		g = strings.ToLower(strings.TrimSpace(g))
		if _, ok := validGenders[g]; !ok {
			return ErrInvalidInput
		}
		if _, dup := seen[g]; !dup {
			seen[g] = struct{}{}
			out = append(out, g)
		}
	}
	s.InterestedIn = out
	if s.MinAge < MinAge || s.MaxAge > MaxAge || s.MinAge > s.MaxAge {
		return ErrInvalidInput
	}
	if s.MaxDistanceKm < 1 || s.MaxDistanceKm > 500 {
		return ErrInvalidInput
	}
	return nil
}
