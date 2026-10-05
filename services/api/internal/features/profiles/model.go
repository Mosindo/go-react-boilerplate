package profiles

import "time"

const (
	MinAge        = 18
	MaxAge        = 120
	MaxInterests  = 10
	MaxPhotos     = 6
	MaxDistanceKm = 500
)

var Genders = []string{"woman", "man", "non_binary"}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

// Card is the public view of a profile shown to other users. It never carries coordinates or birth date.
type Card struct {
	ID         string     `json:"id"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	Occupation string     `json:"occupation"`
	City       string     `json:"city,omitempty"`
	DistanceKm *int       `json:"distanceKm,omitempty"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
}

type OwnProfile struct {
	ID             string     `json:"id"`
	FirstName      string     `json:"firstName"`
	BirthDate      string     `json:"birthDate"`
	Age            int        `json:"age"`
	Gender         string     `json:"gender"`
	Bio            string     `json:"bio"`
	Occupation     string     `json:"occupation"`
	City           string     `json:"city"`
	HasLocation    bool       `json:"hasLocation"`
	IsDiscoverable bool       `json:"isDiscoverable"`
	ShowDistance   bool       `json:"showDistance"`
	Interests      []Interest `json:"interests"`
	Photos         []Photo    `json:"photos"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// Completion tells the client which onboarding steps remain. Only complete profiles can use discovery.
type Completion struct {
	Profile     bool `json:"profile"`
	Preferences bool `json:"preferences"`
	Location    bool `json:"location"`
	Photos      bool `json:"photos"`
	Complete    bool `json:"complete"`
}

type MyProfileResponse struct {
	Profile     *OwnProfile  `json:"profile"`
	Preferences *Preferences `json:"preferences"`
	Completion  Completion   `json:"completion"`
}

type UpsertProfileRequest struct {
	FirstName  string   `json:"firstName" binding:"required,max=80"`
	BirthDate  string   `json:"birthDate" binding:"required,max=10"`
	Gender     string   `json:"gender" binding:"required,max=20"`
	Bio        string   `json:"bio" binding:"max=2000"`
	Occupation string   `json:"occupation" binding:"max=240"`
	Interests  []string `json:"interests" binding:"max=50,dive,max=40"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,min=1,max=3,dive,max=20"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      string   `json:"city" binding:"max=200"`
}

type PrivacyRequest struct {
	IsDiscoverable *bool `json:"isDiscoverable" binding:"required"`
	ShowDistance   *bool `json:"showDistance" binding:"required"`
}
