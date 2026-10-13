package profiles

import "time"

const (
	MinAge          = 18
	MaxAge          = 99
	MaxBioRunes     = 500
	MaxNameRunes    = 40
	MaxCityRunes    = 80
	MaxInterests    = 10
	MaxDistanceKm   = 500
	LocationDecimal = 100.0 // coordinates are rounded to 0.01 degree (~1 km)
)

var Genders = []string{"woman", "man", "nonbinary"}

type Interest struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type PhotoRef struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
}

// OwnProfile is what a user sees about themselves.
type OwnProfile struct {
	ID           string       `json:"id"`
	FirstName    string       `json:"firstName"`
	BirthDate    string       `json:"birthDate"`
	Age          int          `json:"age"`
	Gender       string       `json:"gender"`
	Bio          string       `json:"bio"`
	City         *string      `json:"city"`
	Latitude     *float64     `json:"latitude"`
	Longitude    *float64     `json:"longitude"`
	IsVisible    bool         `json:"isVisible"`
	ShowDistance bool         `json:"showDistance"`
	InterestIDs  []int        `json:"interestIds"`
	Photos       []PhotoRef   `json:"photos"`
	Preferences  *Preferences `json:"preferences"`
	IsComplete   bool         `json:"isComplete"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

// PublicProfile is what other users see. It never carries coordinates, birth
// date, email or visibility settings.
type PublicProfile struct {
	ID         string     `json:"id"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       *string    `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	Photos     []PhotoRef `json:"photos"`
}

type ProfileRequest struct {
	FirstName    string  `json:"firstName"`
	BirthDate    string  `json:"birthDate"`
	Gender       string  `json:"gender"`
	Bio          string  `json:"bio"`
	City         *string `json:"city"`
	InterestIDs  []int   `json:"interestIds"`
	IsVisible    *bool   `json:"isVisible"`
	ShowDistance *bool   `json:"showDistance"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

// ProfileInput is the validated, normalized form persisted by the repository.
type ProfileInput struct {
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	City         *string
	InterestIDs  []int
	IsVisible    bool
	ShowDistance bool
}
