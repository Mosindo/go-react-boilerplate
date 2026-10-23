package profiles

import (
	"time"

	"example.com/api/internal/features/photos"
)

var validGenders = map[string]struct{}{"man": {}, "woman": {}, "non_binary": {}}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type ProfileInput struct {
	FirstName     string   `json:"firstName"`
	BirthDate     string   `json:"birthDate"`
	Gender        string   `json:"gender"`
	Bio           string   `json:"bio"`
	City          string   `json:"city"`
	Latitude      *float64 `json:"latitude"`
	Longitude     *float64 `json:"longitude"`
	ClearLocation bool     `json:"clearLocation"`
	Interests     []string `json:"interests"`
}

// StoredProfile is the validated, normalized form persisted by the repository.
type StoredProfile struct {
	UserID    string
	FirstName string
	BirthDate time.Time
	Gender    string
	Bio       string
	City      string
	Latitude  *float64
	Longitude *float64
}

type Profile struct {
	UserID      string         `json:"userId"`
	FirstName   string         `json:"firstName"`
	BirthDate   string         `json:"birthDate"`
	Age         int            `json:"age"`
	Gender      string         `json:"gender"`
	Bio         string         `json:"bio"`
	City        string         `json:"city"`
	HasLocation bool           `json:"hasLocation"`
	Interests   []Interest     `json:"interests"`
	Photos      []photos.Photo `json:"photos"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
	Discoverable  bool     `json:"discoverable"`
	ShowDistance  bool     `json:"showDistance"`
}
