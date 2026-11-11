package profiles

import "time"

var Genders = []string{"woman", "man", "nonbinary"}

type Profile struct {
	UserID            string
	FirstName         string
	Birthdate         time.Time
	Gender            string
	Bio               string
	JobTitle          string
	City              string
	Latitude          *float64
	Longitude         *float64
	LocationUpdatedAt *time.Time
	Discoverable      bool
	ShowDistance      bool
}

type Preferences struct {
	InterestedIn  []string
	AgeMin        int
	AgeMax        int
	MaxDistanceKm int
}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
	Width      int
	Height     int
}

// --- requests ---

type ProfileRequest struct {
	FirstName string `json:"firstName" binding:"required,max=80"`
	Birthdate string `json:"birthdate" binding:"required,len=10"`
	Gender    string `json:"gender" binding:"required,max=16"`
	Bio       string `json:"bio" binding:"max=1000"`
	JobTitle  string `json:"jobTitle" binding:"max=120"`
	City      string `json:"city" binding:"max=120"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,min=1,max=3"`
	AgeMin        int      `json:"ageMin" binding:"required"`
	AgeMax        int      `json:"ageMax" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type InterestsRequest struct {
	Interests []string `json:"interests" binding:"max=20"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
}

type VisibilityRequest struct {
	Discoverable *bool `json:"discoverable"`
	ShowDistance *bool `json:"showDistance"`
}

// --- responses ---

type PhotoResponse struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type ProfileResponse struct {
	FirstName string `json:"firstName"`
	Birthdate string `json:"birthdate"`
	Age       int    `json:"age"`
	Gender    string `json:"gender"`
	Bio       string `json:"bio"`
	JobTitle  string `json:"jobTitle"`
	City      string `json:"city"`
}

type PreferencesResponse struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type MyProfileResponse struct {
	Profile           *ProfileResponse     `json:"profile"`
	Preferences       *PreferencesResponse `json:"preferences"`
	Interests         []Interest           `json:"interests"`
	Photos            []PhotoResponse      `json:"photos"`
	HasLocation       bool                 `json:"hasLocation"`
	LocationUpdatedAt *time.Time           `json:"locationUpdatedAt"`
	Discoverable      bool                 `json:"discoverable"`
	ShowDistance      bool                 `json:"showDistance"`
	IsComplete        bool                 `json:"isComplete"`
}

type CardInterest struct {
	Slug   string `json:"slug"`
	Label  string `json:"label"`
	Shared bool   `json:"shared"`
}

// Card is the public view of another member. It never contains e-mail,
// birthdate or coordinates.
type Card struct {
	UserID     string          `json:"userId"`
	FirstName  string          `json:"firstName"`
	Age        int             `json:"age"`
	Gender     string          `json:"gender"`
	Bio        string          `json:"bio"`
	JobTitle   string          `json:"jobTitle"`
	City       string          `json:"city"`
	DistanceKm *int            `json:"distanceKm"`
	Interests  []CardInterest  `json:"interests"`
	Photos     []PhotoResponse `json:"photos"`
}

type InterestsCatalogResponse struct {
	Interests []Interest `json:"interests"`
}
