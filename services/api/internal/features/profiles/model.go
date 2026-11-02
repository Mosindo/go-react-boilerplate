package profiles

import "time"

const (
	GenderWoman     = "woman"
	GenderMan       = "man"
	GenderNonBinary = "non_binary"

	MinAge       = 18
	MaxAge       = 99
	MaxBio       = 500
	MaxName      = 40
	MaxCity      = 80
	MaxInterests = 10
)

var validGenders = map[string]bool{GenderWoman: true, GenderMan: true, GenderNonBinary: true}

type Interest struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type PhotoRef struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

// Profile is the owner's own view of their profile.
type Profile struct {
	UserID       string     `json:"userId"`
	FirstName    string     `json:"firstName"`
	BirthDate    string     `json:"birthDate"`
	Age          int        `json:"age"`
	Gender       string     `json:"gender"`
	Bio          string     `json:"bio"`
	City         string     `json:"city"`
	Latitude     *float64   `json:"latitude"`
	Longitude    *float64   `json:"longitude"`
	ShowDistance bool       `json:"showDistance"`
	Discoverable bool       `json:"discoverable"`
	Interests    []Interest `json:"interests"`
	Photos       []PhotoRef `json:"photos"`
	CreatedAt    time.Time  `json:"createdAt"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// Bundle is what GET /me/profile returns. Complete means the user can appear
// in discovery: profile, preferences, location and at least one photo.
type Bundle struct {
	Profile     *Profile     `json:"profile"`
	Preferences *Preferences `json:"preferences"`
	Complete    bool         `json:"complete"`
}

type UpsertProfileRequest struct {
	FirstName    string   `json:"firstName"`
	BirthDate    string   `json:"birthDate"`
	Gender       string   `json:"gender"`
	Bio          string   `json:"bio"`
	City         string   `json:"city"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	ShowDistance *bool    `json:"showDistance"`
	Discoverable *bool    `json:"discoverable"`
	InterestIDs  []int    `json:"interestIds"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}
