package profiles

import "time"

const (
	GenderWoman     = "woman"
	GenderMan       = "man"
	GenderNonbinary = "nonbinary"

	MinAge          = 18
	MaxAge          = 99
	MaxInterests    = 10
	MaxBioRunes     = 500
	MaxNameRunes    = 40
	MaxCityRunes    = 80
	MinDistanceKm   = 1
	MaxDistanceKm   = 300
	DefaultDistance = 50
)

var Genders = []string{GenderWoman, GenderMan, GenderNonbinary}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type Settings struct {
	IsDiscoverable bool `json:"isDiscoverable"`
	HideDistance   bool `json:"hideDistance"`
}

// StoredProfile mirrors the profiles row.
type StoredProfile struct {
	UserID         string
	FirstName      *string
	Birthdate      *time.Time
	Gender         *string
	Bio            string
	City           *string
	Latitude       *float64
	Longitude      *float64
	IsDiscoverable bool
	HideDistance   bool
}

// MyProfile is the owner's complete view of their profile.
type MyProfile struct {
	UserID       string      `json:"userId"`
	FirstName    string      `json:"firstName"`
	Birthdate    *string     `json:"birthdate"`
	Age          *int        `json:"age"`
	Gender       string      `json:"gender"`
	Bio          string      `json:"bio"`
	City         string      `json:"city"`
	HasLocation  bool        `json:"hasLocation"`
	Interests    []Interest  `json:"interests"`
	Photos       []Photo     `json:"photos"`
	Preferences  Preferences `json:"preferences"`
	Settings     Settings    `json:"settings"`
	Missing      []string    `json:"missing"`
	IsComplete   bool        `json:"isComplete"`
	MaxPhotos    int         `json:"maxPhotos"`
	MaxInterests int         `json:"maxInterests"`
}

// PublicProfile is what other members can see. It never contains exact
// coordinates, the birthdate or the email address.
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

type UpdateProfileRequest struct {
	FirstName *string   `json:"firstName"`
	Birthdate *string   `json:"birthdate"`
	Gender    *string   `json:"gender"`
	Bio       *string   `json:"bio"`
	City      *string   `json:"city"`
	Interests *[]string `json:"interests"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type UpdateSettingsRequest struct {
	IsDiscoverable *bool `json:"isDiscoverable" binding:"required"`
	HideDistance   *bool `json:"hideDistance" binding:"required"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      *string  `json:"city"`
}

// ProfileUpdate is a validated partial update.
type ProfileUpdate struct {
	FirstName *string
	Birthdate *time.Time
	Gender    *string
	Bio       *string
	City      *string
	Interests *[]string
}

// CardRow is the batched data needed to render public profiles.
type CardRow struct {
	UserID       string
	FirstName    string
	Birthdate    time.Time
	Gender       string
	Bio          string
	City         string
	HideDistance bool
	DistanceKm   *float64
}

type PhotoRow struct {
	ID       string
	UserID   string
	Position int
}

type InterestRow struct {
	UserID string
	Interest
}
