package profiles

import "time"

const (
	MinAge        = 18
	MaxAge        = 120
	MaxPhotos     = 6
	MaxInterests  = 10
	MaxBioRunes   = 500
	MaxNameRunes  = 40
	MaxCityRunes  = 80
	GenderMan     = "man"
	GenderWoman   = "woman"
	GenderNonBin  = "non_binary"
	DefaultMaxKm  = 50
	DefaultMaxAge = 99
)

var Genders = []string{GenderMan, GenderWoman, GenderNonBin}

type Interest struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type PhotoRef struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// SelfProfile is what a user sees about their own account (includes private settings).
type SelfProfile struct {
	UserID       string      `json:"userId"`
	Exists       bool        `json:"exists"`
	FirstName    string      `json:"firstName"`
	BirthDate    string      `json:"birthDate"`
	Age          int         `json:"age"`
	Gender       string      `json:"gender"`
	Bio          string      `json:"bio"`
	City         string      `json:"city"`
	HasLocation  bool        `json:"hasLocation"`
	IsVisible    bool        `json:"isVisible"`
	ShowDistance bool        `json:"showDistance"`
	Interests    []Interest  `json:"interests"`
	Photos       []PhotoRef  `json:"photos"`
	Preferences  Preferences `json:"preferences"`
	Complete     bool        `json:"complete"`
	Missing      []string    `json:"missing"`
}

// Card is the public view of another user. It never contains coordinates, birth date or email.
type Card struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	Photos     []PhotoRef `json:"photos"`
}

type ProfileInput struct {
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	City         string
	IsVisible    *bool
	ShowDistance *bool
}

type UpsertProfileRequest struct {
	FirstName    string `json:"firstName" binding:"required"`
	BirthDate    string `json:"birthDate" binding:"required"`
	Gender       string `json:"gender" binding:"required"`
	Bio          string `json:"bio"`
	City         string `json:"city"`
	IsVisible    *bool  `json:"isVisible"`
	ShowDistance *bool  `json:"showDistance"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type LocationRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
	City      string  `json:"city"`
}

type InterestsRequest struct {
	InterestIDs []int `json:"interestIds" binding:"required"`
}
