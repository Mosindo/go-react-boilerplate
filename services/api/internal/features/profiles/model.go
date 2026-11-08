package profiles

import "time"

const (
	MaxPhotos    = 6
	MaxInterests = 8
)

var Genders = []string{"man", "woman", "non_binary"}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type PhotoRef struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

// Card is the public view of a profile as seen by another user. It never contains
// coordinates, e-mail or birth date, only an approximate distance and the age.
type Card struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender,omitempty"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm,omitempty"`
	Interests  []Interest `json:"interests"`
	Photos     []PhotoRef `json:"photos"`
}

// Me is the private view of the caller's own profile.
type Me struct {
	Card
	BirthDate    string `json:"birthDate"`
	Discoverable bool   `json:"discoverable"`
	ShowDistance bool   `json:"showDistance"`
	HasLocation  bool   `json:"hasLocation"`
	Complete     bool   `json:"complete"`
	// Missing lists what is still required before the profile appears in discovery.
	Missing []string `json:"missing"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type UpdateProfileRequest struct {
	FirstName    *string   `json:"firstName"`
	Gender       *string   `json:"gender"`
	Bio          *string   `json:"bio"`
	City         *string   `json:"city"`
	Interests    *[]string `json:"interests"` // slugs
	Discoverable *bool     `json:"discoverable"`
	ShowDistance *bool     `json:"showDistance"`
}

type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      *string `json:"city"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type storedProfile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Gender       *string
	Bio          string
	City         string
	Discoverable bool
	ShowDistance bool
	HasLocation  bool
	DistanceKm   *int
	PhotoCount   int
	InterestedIn []string
}
