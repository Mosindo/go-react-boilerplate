package profiles

import "time"

const (
	MinAge = 18
	MaxAge = 99
)

var (
	validGenders = map[string]bool{"man": true, "woman": true, "nonbinary": true}
	maxInterests = 10
)

type Interest struct {
	ID    int    `json:"-"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

// Profile is the owner's view of a profile (includes the birth date).
type Profile struct {
	FirstName   string     `json:"firstName"`
	BirthDate   string     `json:"birthDate"`
	Age         int        `json:"age"`
	Gender      string     `json:"gender"`
	Bio         string     `json:"bio"`
	City        string     `json:"city"`
	HasLocation bool       `json:"hasLocation"`
	Interests   []string   `json:"interests"`
	Photos      []PhotoRef `json:"photos"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type PhotoRef struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

type ProfileInput struct {
	FirstName string   `json:"firstName"`
	BirthDate string   `json:"birthDate"`
	Gender    string   `json:"gender"`
	Bio       string   `json:"bio"`
	City      string   `json:"city"`
	Interests []string `json:"interests"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
}

type Privacy struct {
	IsDiscoverable bool `json:"isDiscoverable"`
	ShowDistance   bool `json:"showDistance"`
}

type LocationInput struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Overview is returned by GET /me/profile and drives the onboarding flow.
type Overview struct {
	Profile            *Profile     `json:"profile"`
	Preferences        *Preferences `json:"preferences"`
	Privacy            *Privacy     `json:"privacy"`
	OnboardingComplete bool         `json:"onboardingComplete"`
	Missing            []string     `json:"missing"`
}
