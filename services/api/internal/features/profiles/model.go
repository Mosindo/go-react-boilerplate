package profiles

import "time"

const (
	MinAge       = 18
	MaxAge       = 99
	MaxPhotos    = 6
	MaxInterests = 10
	MaxBioRunes  = 500
	MaxNameRunes = 40
	MaxCityRunes = 80
	AnyDistance  = 20000
)

var validGenders = map[string]bool{"woman": true, "man": true, "nonbinary": true}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type PhotoView struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// MyProfile is what a user sees about themselves. Exact coordinates are never returned.
type MyProfile struct {
	UserID       string      `json:"userId"`
	FirstName    string      `json:"firstName"`
	BirthDate    string      `json:"birthDate"`
	Age          int         `json:"age"`
	Gender       string      `json:"gender"`
	Bio          string      `json:"bio"`
	City         string      `json:"city"`
	HasLocation  bool        `json:"hasLocation"`
	ShowDistance bool        `json:"showDistance"`
	Discoverable bool        `json:"discoverable"`
	Interests    []Interest  `json:"interests"`
	Preferences  Preferences `json:"preferences"`
	Photos       []PhotoView `json:"photos"`
	Complete     bool        `json:"complete"`
}

// Card is what other users see: no birth date, no coordinates, only an approximate distance.
type Card struct {
	UserID          string      `json:"userId"`
	FirstName       string      `json:"firstName"`
	Age             int         `json:"age"`
	Gender          string      `json:"gender"`
	Bio             string      `json:"bio"`
	City            string      `json:"city"`
	DistanceKm      *int        `json:"distanceKm"`
	Photos          []PhotoView `json:"photos"`
	Interests       []Interest  `json:"interests"`
	SharedInterests int         `json:"sharedInterests"`
}

// UpsertProfileRequest replaces the whole profile (identity, location, preferences, interests).
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
	Interests    []string `json:"interests"`
	Preferences  struct {
		InterestedIn  []string `json:"interestedIn"`
		AgeMin        int      `json:"ageMin"`
		AgeMax        int      `json:"ageMax"`
		MaxDistanceKm int      `json:"maxDistanceKm"`
	} `json:"preferences"`
}

type ReorderPhotosRequest struct {
	IDs []string `json:"ids" binding:"required,max=6,dive,uuid"`
}

// ProfileInput is the validated, normalised form handed to the repository.
type ProfileInput struct {
	FirstName     string
	BirthDate     time.Time
	Gender        string
	Bio           string
	City          string
	Latitude      *float64
	Longitude     *float64
	ShowDistance  bool
	Discoverable  bool
	InterestSlugs []string
	Preferences   Preferences
}

type storedProfile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	City         string
	HasLocation  bool
	ShowDistance bool
	Discoverable bool
}

type storedPhoto struct {
	ID       string
	UserID   string
	Position int
	Key      string
}
