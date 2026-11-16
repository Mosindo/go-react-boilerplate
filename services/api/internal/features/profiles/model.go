package profiles

import "time"

var (
	Genders    = []string{"woman", "man", "nonbinary"}
	LookingFor = []string{"relationship", "casual", "friendship", "unsure"}
)

const (
	MinAge          = 18
	MaxAge          = 99
	MaxInterests    = 10
	MaxBioLength    = 500
	MaxNameLength   = 40
	MaxJobLength    = 60
	MaxCityLength   = 80
	MinDistanceKm   = 1
	MaxDistanceKm   = 500
	DefaultDistance = 50
)

type Interest struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// ProfileInput is the editable part of a profile.
type ProfileInput struct {
	FirstName   string `json:"firstName"`
	Birthdate   string `json:"birthdate"`
	Gender      string `json:"gender"`
	Bio         string `json:"bio"`
	JobTitle    string `json:"jobTitle"`
	HeightCm    *int   `json:"heightCm"`
	LookingFor  string `json:"lookingFor"`
	City        string `json:"city"`
	InterestIDs []int  `json:"interestIds"`
}

type PreferencesInput struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type LocationInput struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
}

type VisibilityInput struct {
	IsDiscoverable *bool `json:"isDiscoverable" binding:"required"`
}

// validated, normalized profile ready for persistence.
type profileRecord struct {
	FirstName   string
	Birthdate   time.Time
	Gender      string
	Bio         string
	JobTitle    string
	HeightCm    *int
	LookingFor  string
	City        string
	InterestIDs []int
}

type MyProfile struct {
	FirstName      string     `json:"firstName"`
	Birthdate      string     `json:"birthdate"`
	Age            int        `json:"age"`
	Gender         string     `json:"gender"`
	Bio            string     `json:"bio"`
	JobTitle       string     `json:"jobTitle"`
	HeightCm       *int       `json:"heightCm"`
	LookingFor     string     `json:"lookingFor"`
	City           string     `json:"city"`
	HasLocation    bool       `json:"hasLocation"`
	IsDiscoverable bool       `json:"isDiscoverable"`
	Interests      []Interest `json:"interests"`
	Photos         []Photo    `json:"photos"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type Completeness struct {
	HasProfile     bool `json:"hasProfile"`
	HasPreferences bool `json:"hasPreferences"`
	HasPhotos      bool `json:"hasPhotos"`
	HasLocation    bool `json:"hasLocation"`
	IsComplete     bool `json:"isComplete"`
}

type MeResponse struct {
	Profile      *MyProfile   `json:"profile"`
	Preferences  *Preferences `json:"preferences"`
	Completeness Completeness `json:"completeness"`
}

// CardInterest flags interests shared with the viewer.
type CardInterest struct {
	ID     int    `json:"id"`
	Label  string `json:"label"`
	Shared bool   `json:"shared"`
}

// Card is the public view of a profile. It never contains email, birthdate
// or coordinates: only an approximate distance.
type Card struct {
	UserID     string         `json:"userId"`
	FirstName  string         `json:"firstName"`
	Age        int            `json:"age"`
	Gender     string         `json:"gender"`
	Bio        string         `json:"bio"`
	JobTitle   string         `json:"jobTitle"`
	HeightCm   *int           `json:"heightCm"`
	LookingFor string         `json:"lookingFor"`
	City       string         `json:"city"`
	DistanceKm *int           `json:"distanceKm"`
	Interests  []CardInterest `json:"interests"`
	Photos     []Photo        `json:"photos"`
}
