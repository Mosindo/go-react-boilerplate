package profiles

import "time"

// Valid enumerations, mirrored by CHECK constraints in the database.
var Genders = []string{"man", "woman", "non_binary"}

const (
	MinAge          = 18
	MaxAge          = 99
	MaxInterests    = 10
	MaxBioRunes     = 500
	MaxNameRunes    = 40
	MaxCityRunes    = 80
	DefaultDistance = 50
	MaxDistanceKm   = 500
)

type Interest struct {
	ID    int16  `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type PhotoRef struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	// Version changes when the image is replaced; clients append it to bust caches.
	Version string `json:"version"`
}

// SelfProfile is what the owner sees, including the raw birthdate and coarse coordinates.
type SelfProfile struct {
	UserID         string     `json:"userId"`
	FirstName      string     `json:"firstName"`
	Birthdate      string     `json:"birthdate"`
	Age            int        `json:"age"`
	Gender         string     `json:"gender"`
	Bio            string     `json:"bio"`
	City           string     `json:"city"`
	Latitude       float64    `json:"latitude"`
	Longitude      float64    `json:"longitude"`
	IsDiscoverable bool       `json:"isDiscoverable"`
	ShowDistance   bool       `json:"showDistance"`
	Interests      []Interest `json:"interests"`
	Photos         []PhotoRef `json:"photos"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// PublicProfile is what other users see: no birthdate, no coordinates, approximate distance only.
type PublicProfile struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm,omitempty"`
	Interests  []Interest `json:"interests"`
	Photos     []PhotoRef `json:"photos"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type UpdateProfileRequest struct {
	FirstName      string  `json:"firstName"`
	Birthdate      string  `json:"birthdate"`
	Gender         string  `json:"gender"`
	Bio            string  `json:"bio"`
	City           string  `json:"city"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	InterestIDs    []int16 `json:"interestIds"`
	IsDiscoverable *bool   `json:"isDiscoverable"`
	ShowDistance   *bool   `json:"showDistance"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// StoredProfile is the validated, normalised write model.
type StoredProfile struct {
	UserID         string
	FirstName      string
	Birthdate      time.Time
	Gender         string
	Bio            string
	City           string
	Latitude       float64
	Longitude      float64
	IsDiscoverable bool
	ShowDistance   bool
	InterestIDs    []int16
}
