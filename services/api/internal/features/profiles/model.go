package profiles

import "time"

var validGenders = map[string]struct{}{"woman": {}, "man": {}, "non_binary": {}, "other": {}}

const (
	MinAge          = 18
	MaxAge          = 99
	maxInterests    = 10
	maxFirstNameLen = 50
	maxBioLen       = 500
	maxCityLen      = 80
	maxDistanceKm   = 20000
	defaultDistance = 50
)

type InterestRef struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type PhotoRef struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

// PublicProfile is what other users may see. It never contains email,
// birth date, coordinates or any other private field.
type PublicProfile struct {
	UserID     string        `json:"id"`
	FirstName  string        `json:"firstName"`
	Age        int           `json:"age"`
	Gender     string        `json:"gender"`
	Bio        string        `json:"bio"`
	City       string        `json:"city"`
	DistanceKm *int          `json:"distanceKm"`
	Interests  []InterestRef `json:"interests"`
	Photos     []PhotoRef    `json:"photos"`
}

// OwnProfile is the owner's view of their profile.
type OwnProfile struct {
	UserID       string        `json:"id"`
	FirstName    string        `json:"firstName"`
	BirthDate    string        `json:"birthDate"`
	Age          int           `json:"age"`
	Gender       string        `json:"gender"`
	Bio          string        `json:"bio"`
	City         string        `json:"city"`
	HasLocation  bool          `json:"hasLocation"`
	Discoverable bool          `json:"discoverable"`
	ShowDistance bool          `json:"showDistance"`
	Interests    []InterestRef `json:"interests"`
	Photos       []PhotoRef    `json:"photos"`
	Complete     bool          `json:"complete"`
	CreatedAt    time.Time     `json:"createdAt"`
	UpdatedAt    time.Time     `json:"updatedAt"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// UpsertProfileRequest creates or updates the caller's profile.
// birthDate is required on creation and immutable afterwards.
type UpsertProfileRequest struct {
	FirstName    string   `json:"firstName"`
	BirthDate    string   `json:"birthDate"`
	Gender       string   `json:"gender"`
	Bio          string   `json:"bio"`
	City         string   `json:"city"`
	Interests    []string `json:"interests"`
	Discoverable *bool    `json:"discoverable"`
	ShowDistance *bool    `json:"showDistance"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	City      string   `json:"city"`
}

// ProfileInput is the validated, normalized form handed to the repository.
type ProfileInput struct {
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	City         string
	Interests    []string
	Discoverable *bool
	ShowDistance *bool
}
