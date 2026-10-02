package profiles

import (
	"time"

	"example.com/api/internal/features/photos"
)

const (
	MinAge         = 18
	MaxAge         = 99
	MaxInterests   = 10
	MaxBioLen      = 500
	MaxFirstName   = 50
	MaxCityLen     = 80
	coordPrecision = 100 // 2 decimals ~ 1.1 km
)

var Genders = []string{"woman", "man", "nonbinary"}

type Interest struct {
	Slug   string `json:"slug"`
	Label  string `json:"label"`
	Shared bool   `json:"shared,omitempty"`
}

// Card is the public view of a profile. It never carries coordinates or birth date.
type Card struct {
	ID         string        `json:"id"`
	FirstName  string        `json:"firstName"`
	Age        int           `json:"age"`
	Gender     string        `json:"gender"`
	Bio        string        `json:"bio"`
	City       string        `json:"city"`
	DistanceKm *int          `json:"distanceKm,omitempty"`
	Interests  []Interest    `json:"interests"`
	Photos     []photos.View `json:"photos"`
}

// Own is the owner's complete view of their profile.
type Own struct {
	FirstName    string        `json:"firstName"`
	BirthDate    string        `json:"birthDate"`
	Age          int           `json:"age"`
	Gender       string        `json:"gender"`
	Bio          string        `json:"bio"`
	City         string        `json:"city"`
	Latitude     float64       `json:"latitude"`
	Longitude    float64       `json:"longitude"`
	Discoverable bool          `json:"discoverable"`
	ShowDistance bool          `json:"showDistance"`
	Interests    []Interest    `json:"interests"`
	Photos       []photos.View `json:"photos"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,min=1,max=3"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
}

// Status tells the client where the user is in onboarding.
type Status struct {
	Profile     *Own         `json:"profile"`
	Preferences *Preferences `json:"preferences"`
	Complete    bool         `json:"complete"`
	Missing     []string     `json:"missing"`
}

type ProfileInput struct {
	FirstName    string   `json:"firstName" binding:"required,max=100"`
	BirthDate    string   `json:"birthDate" binding:"required,max=10"`
	Gender       string   `json:"gender" binding:"required,max=20"`
	Bio          string   `json:"bio" binding:"max=2000"`
	City         string   `json:"city" binding:"max=200"`
	Latitude     float64  `json:"latitude"`
	Longitude    float64  `json:"longitude"`
	Interests    []string `json:"interests" binding:"max=50"`
	Discoverable *bool    `json:"discoverable"`
	ShowDistance *bool    `json:"showDistance"`
}

type StoredProfile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	City         string
	Latitude     float64
	Longitude    float64
	Discoverable bool
	ShowDistance bool
	Age          int
}
