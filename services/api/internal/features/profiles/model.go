package profiles

import "time"

const (
	GenderWoman     = "woman"
	GenderMan       = "man"
	GenderNonBinary = "nonbinary"

	MinAge       = 18
	MaxAge       = 99
	MaxInterests = 10
)

var validGenders = map[string]struct{}{GenderWoman: {}, GenderMan: {}, GenderNonBinary: {}}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type PhotoRef struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
}

// Card is the public face of a profile shown to other users. It never carries
// email, birth date or coordinates.
type Card struct {
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
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
}

// Own is the full private view of the caller's profile.
type Own struct {
	UserID       string       `json:"userId"`
	FirstName    string       `json:"firstName"`
	BirthDate    string       `json:"birthDate"`
	Age          int          `json:"age"`
	Gender       string       `json:"gender"`
	Bio          string       `json:"bio"`
	City         string       `json:"city"`
	HasLocation  bool         `json:"hasLocation"`
	Discoverable bool         `json:"discoverable"`
	ShowDistance bool         `json:"showDistance"`
	Interests    []Interest   `json:"interests"`
	Photos       []PhotoRef   `json:"photos"`
	Preferences  *Preferences `json:"preferences"`
	Complete     bool         `json:"complete"`
}

type UpsertProfileRequest struct {
	FirstName    string   `json:"firstName"`
	BirthDate    string   `json:"birthDate"` // YYYY-MM-DD
	Gender       string   `json:"gender"`
	Bio          string   `json:"bio"`
	City         string   `json:"city"`
	Interests    []string `json:"interests"`
	Discoverable *bool    `json:"discoverable"`
	ShowDistance *bool    `json:"showDistance"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
}

// stored shapes (repository <-> service)

type storedProfile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	City         string
	Latitude     *float64
	Longitude    *float64
	Discoverable bool
	ShowDistance bool
}

type ProfileInput struct {
	FirstName     string
	BirthDate     time.Time
	Gender        string
	Bio           string
	City          string
	InterestSlugs []string
	Discoverable  bool
	ShowDistance  bool
}
