package profiles

import "time"

const (
	MinAge          = 18
	MaxAge          = 99
	MaxInterests    = 10
	MaxPhotos       = 6
	DefaultDistance = 50
)

var (
	Genders     = []string{"woman", "man", "nonbinary"}
	LookingFors = []string{"relationship", "casual", "friendship", "unsure"}
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

type Preferences struct {
	InterestedIn   []string `json:"interestedIn"`
	AgeMin         int      `json:"ageMin"`
	AgeMax         int      `json:"ageMax"`
	MaxDistanceKm  int      `json:"maxDistanceKm"`
	IsDiscoverable bool     `json:"isDiscoverable"`
	ShowDistance   bool     `json:"showDistance"`
}

// StoredProfile is the full private record of a profile.
type StoredProfile struct {
	UserID     string
	FirstName  string
	Birthdate  time.Time
	Gender     string
	Bio        string
	JobTitle   string
	LookingFor string
	City       string
	Latitude   *float64
	Longitude  *float64
}

// OwnProfile is what a user sees about themself.
type OwnProfile struct {
	UserID      string      `json:"userId"`
	FirstName   string      `json:"firstName"`
	Birthdate   string      `json:"birthdate"`
	Age         int         `json:"age"`
	Gender      string      `json:"gender"`
	Bio         string      `json:"bio"`
	JobTitle    string      `json:"jobTitle"`
	LookingFor  string      `json:"lookingFor"`
	City        string      `json:"city"`
	HasLocation bool        `json:"hasLocation"`
	Interests   []Interest  `json:"interests"`
	Photos      []Photo     `json:"photos"`
	IsComplete  bool        `json:"isComplete"`
	Preferences Preferences `json:"preferences"`
}

// Card is the public, privacy-safe view of another user's profile.
type Card struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	JobTitle   string     `json:"jobTitle"`
	LookingFor string     `json:"lookingFor"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
}

type UpsertProfileRequest struct {
	FirstName   string `json:"firstName" binding:"required"`
	Birthdate   string `json:"birthdate" binding:"required"`
	Gender      string `json:"gender" binding:"required"`
	Bio         string `json:"bio"`
	JobTitle    string `json:"jobTitle"`
	LookingFor  string `json:"lookingFor"`
	City        string `json:"city"`
	InterestIDs []int  `json:"interestIds"`
}

type UpdatePreferencesRequest struct {
	InterestedIn   []string `json:"interestedIn" binding:"required"`
	AgeMin         int      `json:"ageMin" binding:"required"`
	AgeMax         int      `json:"ageMax" binding:"required"`
	MaxDistanceKm  int      `json:"maxDistanceKm" binding:"required"`
	IsDiscoverable *bool    `json:"isDiscoverable" binding:"required"`
	ShowDistance   *bool    `json:"showDistance" binding:"required"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      *string  `json:"city"`
}
