package profiles

import "time"

const (
	GenderWoman     = "woman"
	GenderMan       = "man"
	GenderNonBinary = "nonbinary"

	MinAge = 18
	MaxAge = 99
)

var validGenders = map[string]bool{GenderWoman: true, GenderMan: true, GenderNonBinary: true}

// Photo is a photo as exposed by the API (URL is signed and expiring).
type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

// Preferences are the dating preferences of a user.
type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// OwnProfile is the private view of one's own profile.
type OwnProfile struct {
	UserID      string       `json:"userId"`
	FirstName   string       `json:"firstName"`
	BirthDate   string       `json:"birthDate"`
	Age         int          `json:"age"`
	Gender      string       `json:"gender"`
	Bio         string       `json:"bio"`
	City        string       `json:"city"`
	HasLocation bool         `json:"hasLocation"`
	IsVisible   bool         `json:"isVisible"`
	Interests   []Interest   `json:"interests"`
	Photos      []Photo      `json:"photos"`
	Preferences *Preferences `json:"preferences"`
	IsComplete  bool         `json:"isComplete"`
	Missing     []string     `json:"missing"`
}

// PublicProfile is what other users can see. It never contains coordinates,
// email or birth date.
type PublicProfile struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
}

// UpsertProfileRequest creates or updates the profile. birthDate is immutable once set.
type UpsertProfileRequest struct {
	FirstName string    `json:"firstName"`
	BirthDate *string   `json:"birthDate"`
	Gender    string    `json:"gender"`
	Bio       *string   `json:"bio"`
	City      *string   `json:"city"`
	Interests *[]string `json:"interests"`
	IsVisible *bool     `json:"isVisible"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type LocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      *string `json:"city"`
}

type storedProfile struct {
	UserID      string
	FirstName   string
	BirthDate   time.Time
	Gender      string
	Bio         string
	City        string
	HasLocation bool
	IsVisible   bool
}

type storedPhoto struct {
	ID       string
	UserID   string
	Position int
}
