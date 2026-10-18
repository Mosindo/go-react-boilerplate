package profiles

import (
	"time"

	"example.com/api/internal/features/photos"
)

const (
	MinAge        = 18
	MaxAge        = 99
	MaxInterests  = 8
	MaxBioRunes   = 500
	MaxNameRunes  = 40
	MaxCityRunes  = 80
	MaxDistanceKm = 20000
)

var Genders = []string{"man", "woman", "nonbinary"}

type InterestRef struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,min=1,max=3,dive,oneof=man woman nonbinary"`
	AgeMin        int      `json:"ageMin" binding:"required,min=18,max=99"`
	AgeMax        int      `json:"ageMax" binding:"required,min=18,max=99"`
	MaxDistanceKm *int     `json:"maxDistanceKm" binding:"omitempty,min=1,max=20000"`
}

// UpsertRequest creates or replaces the caller's whole profile. Pointers mark
// fields where the zero value is meaningful (0,0 is a valid coordinate).
type UpsertRequest struct {
	FirstName   string      `json:"firstName" binding:"required,max=200"`
	BirthDate   string      `json:"birthDate" binding:"required,max=10"`
	Gender      string      `json:"gender" binding:"required,oneof=man woman nonbinary"`
	Bio         string      `json:"bio" binding:"max=4000"`
	City        string      `json:"city" binding:"max=400"`
	Latitude    *float64    `json:"latitude" binding:"required,min=-90,max=90"`
	Longitude   *float64    `json:"longitude" binding:"required,min=-180,max=180"`
	Interests   []string    `json:"interests" binding:"max=40,dive,max=40"`
	Preferences Preferences `json:"preferences" binding:"required"`
}

type SettingsRequest struct {
	IsVisible    *bool `json:"isVisible"`
	ShowDistance *bool `json:"showDistance"`
}

// MyProfile is the owner's complete view of their own profile.
type MyProfile struct {
	UserID       string         `json:"userId"`
	FirstName    string         `json:"firstName"`
	BirthDate    string         `json:"birthDate"`
	Age          int            `json:"age"`
	Gender       string         `json:"gender"`
	Bio          string         `json:"bio"`
	City         string         `json:"city"`
	Latitude     float64        `json:"latitude"`
	Longitude    float64        `json:"longitude"`
	IsVisible    bool           `json:"isVisible"`
	ShowDistance bool           `json:"showDistance"`
	Interests    []InterestRef  `json:"interests"`
	Preferences  Preferences    `json:"preferences"`
	Photos       []photos.Photo `json:"photos"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

// PublicProfile is what other users may see. It never carries coordinates,
// birth date, email or visibility flags: only an age and a coarse distance.
type PublicProfile struct {
	UserID     string         `json:"userId"`
	FirstName  string         `json:"firstName"`
	Age        int            `json:"age"`
	Gender     string         `json:"gender"`
	Bio        string         `json:"bio"`
	City       string         `json:"city"`
	DistanceKm *int           `json:"distanceKm"`
	Interests  []InterestRef  `json:"interests"`
	Photos     []photos.Photo `json:"photos"`
}

// Card is the raw row other features (discovery, matches) load before hydration.
type Card struct {
	UserID     string
	FirstName  string
	BirthDate  time.Time
	Gender     string
	Bio        string
	City       string
	DistanceKm *float64 // nil when unknown or hidden by the target's privacy setting
}

type InterestsResponse struct {
	Interests []InterestRef `json:"interests"`
}

type storedProfile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	City         string
	Latitude     float64
	Longitude    float64
	IsVisible    bool
	ShowDistance bool
	Preferences  Preferences
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
