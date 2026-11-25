package profiles

import (
	"time"

	"example.com/api/internal/features/photos"
)

var (
	Genders     = []string{"woman", "man", "nonbinary"}
	LookingFors = []string{"relationship", "casual", "friendship", "unsure"}
)

const (
	MinAge           = 18
	MaxAge           = 99
	MaxInterests     = 10
	MaxDistanceKm    = 500
	DefaultDistance  = 50
	birthdateLayout  = "2006-01-02"
	locationDecimals = 100.0 // coordinates are rounded to 0.01° (~1 km)
)

type Interest struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type storedProfile struct {
	FirstName         string
	Birthdate         time.Time
	Gender            string
	Bio               string
	JobTitle          *string
	City              *string
	HeightCm          *int
	LookingFor        *string
	Latitude          *float64
	Longitude         *float64
	LocationUpdatedAt *time.Time
	Discoverable      bool
	ShowDistance      bool
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type Privacy struct {
	Discoverable bool `json:"discoverable"`
	ShowDistance bool `json:"showDistance"`
}

type OwnProfile struct {
	FirstName         string     `json:"firstName"`
	Birthdate         string     `json:"birthdate"`
	Age               int        `json:"age"`
	Gender            string     `json:"gender"`
	Bio               string     `json:"bio"`
	JobTitle          *string    `json:"jobTitle"`
	City              *string    `json:"city"`
	HeightCm          *int       `json:"heightCm"`
	LookingFor        *string    `json:"lookingFor"`
	Interests         []Interest `json:"interests"`
	HasLocation       bool       `json:"hasLocation"`
	LocationUpdatedAt *time.Time `json:"locationUpdatedAt"`
}

type Completeness struct {
	IsComplete bool     `json:"isComplete"`
	Missing    []string `json:"missing"`
}

type MeProfileResponse struct {
	Profile      *OwnProfile    `json:"profile"`
	Preferences  Preferences    `json:"preferences"`
	Privacy      Privacy        `json:"privacy"`
	Photos       []photos.Photo `json:"photos"`
	Completeness Completeness   `json:"completeness"`
}

// CardRow is the raw data needed to render a public profile card. It is
// produced by the profiles repository and by the discovery query.
type CardRow struct {
	UserID       string
	FirstName    string
	Birthdate    time.Time
	Gender       string
	Bio          string
	JobTitle     *string
	City         *string
	HeightCm     *int
	LookingFor   *string
	ShowDistance bool
	DistanceKm   *float64
	IsDemo       bool
}

// PublicProfile is what other members can see. It never contains the email,
// birthdate or coordinates; distance is approximate.
type PublicProfile struct {
	UserID            string         `json:"userId"`
	FirstName         string         `json:"firstName"`
	Age               int            `json:"age"`
	Gender            string         `json:"gender"`
	Bio               string         `json:"bio"`
	JobTitle          *string        `json:"jobTitle"`
	City              *string        `json:"city"`
	HeightCm          *int           `json:"heightCm"`
	LookingFor        *string        `json:"lookingFor"`
	DistanceKm        *int           `json:"distanceKm"`
	Interests         []Interest     `json:"interests"`
	CommonInterestIDs []int          `json:"commonInterestIds"`
	Photos            []photos.Photo `json:"photos"`
	// IsDemo flags seeded demonstration profiles so they are never mistaken
	// for real members.
	IsDemo bool `json:"isDemo"`
}

type UpdateProfileRequest struct {
	FirstName   string  `json:"firstName" binding:"max=200"`
	Birthdate   string  `json:"birthdate" binding:"max=20"`
	Gender      string  `json:"gender" binding:"max=20"`
	Bio         string  `json:"bio" binding:"max=2000"`
	JobTitle    *string `json:"jobTitle" binding:"omitempty,max=200"`
	City        *string `json:"city" binding:"omitempty,max=200"`
	HeightCm    *int    `json:"heightCm"`
	LookingFor  *string `json:"lookingFor" binding:"omitempty,max=20"`
	InterestIDs []int   `json:"interestIds" binding:"max=20"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"max=3"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type UpdatePrivacyRequest struct {
	Discoverable *bool `json:"discoverable" binding:"required"`
	ShowDistance *bool `json:"showDistance" binding:"required"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
}

type InterestsResponse struct {
	Interests []Interest `json:"interests"`
}
