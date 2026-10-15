package users

import (
	"context"
	"time"
)

const (
	GenderMan       = "man"
	GenderWoman     = "woman"
	GenderNonBinary = "nonbinary"

	MinAge             = 18
	MaxAge             = 99
	MaxInterests       = 10
	MaxBioRunes        = 500
	MaxFirstNameRunes  = 40
	MaxCityRunes       = 80
	DefaultMaxDistance = 50
)

var validGenders = map[string]struct{}{
	GenderMan:       {},
	GenderWoman:     {},
	GenderNonBinary: {},
}

// Profile is what a user chooses to show about themselves.
type Profile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	City         *string
	Latitude     *float64
	Longitude    *float64
	IsVisible    bool
	ShowDistance bool
}

type Preferences struct {
	InterestedIn  []string
	MinAge        int
	MaxAge        int
	MaxDistanceKm int
}

type Interest struct {
	ID    int16  `json:"-"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

// PhotoView is the client-facing description of a stored photo.
type PhotoView struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// PhotoSource lets the users module embed photos without depending on the files module.
type PhotoSource interface {
	PhotosForUsers(ctx context.Context, userIDs []string) (map[string][]PhotoView, error)
}

// CardBase is the data a discovery/matching query selects for one person.
type CardBase struct {
	UserID    string
	FirstName string
	BirthDate time.Time
	Gender    string
	Bio       string
	City      *string
	// DistanceKm is the exact distance in kilometres; it is rounded before leaving the service.
	DistanceKm *float64
	LikedMe    bool
}

// Card is the public view of a profile. It never carries exact coordinates.
type Card struct {
	UserID          string      `json:"userId"`
	FirstName       string      `json:"firstName"`
	Age             int         `json:"age"`
	Gender          string      `json:"gender"`
	Bio             string      `json:"bio"`
	City            *string     `json:"city,omitempty"`
	DistanceKm      *int        `json:"distanceKm,omitempty"`
	Interests       []Interest  `json:"interests"`
	CommonInterests int         `json:"commonInterests"`
	Photos          []PhotoView `json:"photos"`
	LikedMe         bool        `json:"likedMe"`
}

type Onboarding struct {
	Profile     bool `json:"profile"`
	Preferences bool `json:"preferences"`
	Photos      bool `json:"photos"`
	Complete    bool `json:"complete"`
}

// ---- HTTP payloads ----

type ProfileInput struct {
	FirstName string   `json:"firstName" binding:"required,max=200"`
	BirthDate string   `json:"birthDate" binding:"required,max=10"`
	Gender    string   `json:"gender" binding:"required,max=16"`
	Bio       string   `json:"bio" binding:"max=2000"`
	Interests []string `json:"interests" binding:"max=50,dive,max=64"`
}

type PreferencesInput struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,max=10,dive,max=16"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type LocationInput struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      string   `json:"city" binding:"max=200"`
}

// PrivacyInput uses pointers so omitted fields keep their current value.
type PrivacyInput struct {
	IsVisible    *bool `json:"isVisible"`
	ShowDistance *bool `json:"showDistance"`
}

type MyProfile struct {
	FirstName    string          `json:"firstName"`
	BirthDate    string          `json:"birthDate"`
	Age          int             `json:"age"`
	Gender       string          `json:"gender"`
	Bio          string          `json:"bio"`
	City         *string         `json:"city,omitempty"`
	HasLocation  bool            `json:"hasLocation"`
	IsVisible    bool            `json:"isVisible"`
	ShowDistance bool            `json:"showDistance"`
	Interests    []Interest      `json:"interests"`
	Photos       []PhotoView     `json:"photos"`
	Preferences  *PreferencesDTO `json:"preferences,omitempty"`
}

type PreferencesDTO struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type MyProfileResponse struct {
	Profile    *MyProfile `json:"profile"`
	Onboarding Onboarding `json:"onboarding"`
}
