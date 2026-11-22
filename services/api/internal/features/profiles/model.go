package profiles

import "time"

const (
	GenderWoman     = "woman"
	GenderMan       = "man"
	GenderNonBinary = "nonbinary"

	MinAge              = 18
	MaxAge              = 99
	MaxInterests        = 10
	MaxPhotos           = 6
	MaxFirstNameRunes   = 40
	MaxBioRunes         = 500
	MaxJobTitleRunes    = 80
	MaxCityRunes        = 80
	MinDistanceKm       = 1
	MaxDistanceKm       = 500
	DefaultDistanceKm   = 50
	birthdateLayout     = "2006-01-02"
	coordinatePrecision = 100.0 // 0.01° ≈ 1.1 km
)

var AllGenders = []string{GenderWoman, GenderMan, GenderNonBinary}

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
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// OwnProfile is what the account owner sees and edits.
type OwnProfile struct {
	UserID           string      `json:"userId"`
	FirstName        string      `json:"firstName"`
	Birthdate        string      `json:"birthdate,omitempty"`
	Age              *int        `json:"age,omitempty"`
	Gender           string      `json:"gender,omitempty"`
	Bio              string      `json:"bio"`
	JobTitle         string      `json:"jobTitle"`
	City             string      `json:"city"`
	HasLocation      bool        `json:"hasLocation"`
	Discoverable     bool        `json:"discoverable"`
	ShowDistance     bool        `json:"showDistance"`
	ModerationStatus string      `json:"moderationStatus"`
	Interests        []Interest  `json:"interests"`
	Photos           []Photo     `json:"photos"`
	Preferences      Preferences `json:"preferences"`
	IsComplete       bool        `json:"isComplete"`
	MissingFields    []string    `json:"missingFields"`
}

// Card is the public, privacy-filtered view of another member.
// It never contains e-mail, birthdate or coordinates.
type Card struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	JobTitle   string     `json:"jobTitle"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm,omitempty"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
	IsDemo     bool       `json:"isDemo"`
}

// ProfilePatch carries optional updates (nil = unchanged).
type ProfilePatch struct {
	FirstName    *string   `json:"firstName"`
	Birthdate    *string   `json:"birthdate"`
	Gender       *string   `json:"gender"`
	Bio          *string   `json:"bio"`
	JobTitle     *string   `json:"jobTitle"`
	Interests    *[]string `json:"interests"`
	Discoverable *bool     `json:"discoverable"`
	ShowDistance *bool     `json:"showDistance"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      string   `json:"city" binding:"max=200"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,max=3"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type InterestsResponse struct {
	Interests []Interest `json:"interests"`
}

// storedProfile mirrors the profiles + preferences rows.
type storedProfile struct {
	UserID           string
	FirstName        *string
	Birthdate        *time.Time
	Gender           *string
	Bio              string
	JobTitle         string
	City             string
	Latitude         *float64
	Longitude        *float64
	Discoverable     bool
	ShowDistance     bool
	ModerationStatus string
	Preferences      Preferences
	PhotoCount       int
}

type storedCard struct {
	UserID       string
	FirstName    string
	Birthdate    time.Time
	Gender       string
	Bio          string
	JobTitle     string
	City         string
	DistanceKm   *float64
	ShowDistance bool
	IsDemo       bool
}

type storedPhoto struct {
	ID       string
	UserID   string
	Position int
	Width    int
	Height   int
}
