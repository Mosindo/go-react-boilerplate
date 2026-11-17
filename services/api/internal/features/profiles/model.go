package profiles

import "time"

var Genders = []string{"woman", "man", "nonbinary"}

const (
	MinAge            = 18
	MaxAge            = 99
	MaxInterests      = 10
	MaxFirstNameRunes = 40
	MaxBioRunes       = 500
	MaxJobTitleRunes  = 80
	MaxCityRunes      = 80
	MinDistanceKm     = 1
	MaxDistanceKm     = 500
	locationPrecision = 100.0 // 2 decimals ≈ 1.1 km
)

type Interest struct {
	ID       int    `json:"id"`
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Category string `json:"category"`
}

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type Settings struct {
	Discoverable bool `json:"discoverable"`
	ShowDistance bool `json:"showDistance"`
}

// StoredProfile is the raw row; coordinates never leave the server.
type StoredProfile struct {
	UserID       string
	FirstName    string
	Birthdate    *time.Time
	Gender       *string
	Bio          string
	JobTitle     string
	City         string
	Latitude     *float64
	Longitude    *float64
	Discoverable bool
	ShowDistance bool
	IsDemo       bool
}

type MyProfile struct {
	UserID      string      `json:"userId"`
	FirstName   string      `json:"firstName"`
	Birthdate   *string     `json:"birthdate"`
	Age         *int        `json:"age"`
	Gender      *string     `json:"gender"`
	Bio         string      `json:"bio"`
	JobTitle    string      `json:"jobTitle"`
	City        string      `json:"city"`
	HasLocation bool        `json:"hasLocation"`
	Settings    Settings    `json:"settings"`
	Preferences Preferences `json:"preferences"`
	Interests   []Interest  `json:"interests"`
	Photos      []Photo     `json:"photos"`
	IsDemo      bool        `json:"isDemo"`
	IsComplete  bool        `json:"isComplete"`
	Missing     []string    `json:"missing"`
}

// PublicProfile is what other members see. Distance is approximate and omitted when
// the owner disabled it; exact coordinates and birthdate are never exposed.
type PublicProfile struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	JobTitle   string     `json:"jobTitle"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
	IsDemo     bool       `json:"isDemo"`
}

type UpdateProfileRequest struct {
	FirstName   *string `json:"firstName"`
	Birthdate   *string `json:"birthdate"`
	Gender      *string `json:"gender"`
	Bio         *string `json:"bio"`
	JobTitle    *string `json:"jobTitle"`
	InterestIDs *[]int  `json:"interestIds"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      *string  `json:"city"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required"`
	AgeMin        int      `json:"ageMin" binding:"required"`
	AgeMax        int      `json:"ageMax" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type UpdateSettingsRequest struct {
	Discoverable *bool `json:"discoverable"`
	ShowDistance *bool `json:"showDistance"`
}
