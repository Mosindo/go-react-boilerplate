package profiles

import "time"

const (
	MinAge         = 18
	MaxAge         = 99
	MaxBioRunes    = 500
	MaxNameRunes   = 40
	MaxCityRunes   = 80
	MaxInterests   = 10
	MaxDistanceKm  = 500
	DefaultMaxDist = 50
)

var Genders = []string{"woman", "man", "nonbinary"}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

// ProfileInput is the validated payload used to create or update a profile.
type ProfileInput struct {
	FirstName string
	BirthDate time.Time
	Gender    string
	Bio       string
	City      string
	Latitude  float64
	Longitude float64
	Interests []string
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// OwnProfile is what a user sees about themselves. Coordinates are never echoed.
type OwnProfile struct {
	FirstName      string    `json:"firstName"`
	BirthDate      string    `json:"birthDate"`
	Age            int       `json:"age"`
	Gender         string    `json:"gender"`
	Bio            string    `json:"bio"`
	City           string    `json:"city"`
	IsDiscoverable bool      `json:"discoverable"`
	ShowDistance   bool      `json:"showDistance"`
	CreatedAt      time.Time `json:"createdAt"`
}

type MyProfileResponse struct {
	Profile            *OwnProfile `json:"profile"`
	Preferences        Preferences `json:"preferences"`
	Interests          []Interest  `json:"interests"`
	Photos             []Photo     `json:"photos"`
	OnboardingComplete bool        `json:"onboardingComplete"`
}

// PublicProfile is what other users see: no email, birth date or coordinates.
type PublicProfile struct {
	ID         string     `json:"id"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
}

type ProfileRequest struct {
	FirstName string   `json:"firstName" binding:"required,max=200"`
	BirthDate string   `json:"birthDate" binding:"required,max=10"`
	Gender    string   `json:"gender" binding:"required,max=20"`
	Bio       string   `json:"bio" binding:"max=2000"`
	City      string   `json:"city" binding:"max=200"`
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	Interests []string `json:"interests" binding:"max=50,dive,max=40"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,max=10,dive,max=20"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type PrivacyRequest struct {
	Discoverable *bool `json:"discoverable" binding:"required"`
	ShowDistance *bool `json:"showDistance" binding:"required"`
}
