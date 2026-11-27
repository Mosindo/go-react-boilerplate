package profiles

import "time"

var AllowedGenders = []string{"woman", "man", "nonbinary"}

const (
	MinAge            = 18
	MaxAge            = 99
	MaxInterests      = 10
	MaxBioRunes       = 500
	MaxFirstNameRunes = 40
	MaxCityRunes      = 80
	MinDistanceKm     = 1
	MaxDistanceKm     = 500
	// Coordinates are snapped to this grid (~1.1 km) before storage so the
	// exact position of a user is never persisted nor inferable.
	LocationGridDegrees = 0.01
)

// Profile is the private, owner-facing profile.
type Profile struct {
	UserID       string
	FirstName    string
	Birthdate    time.Time
	Gender       string
	Bio          string
	City         string
	HasLocation  bool
	IsVisible    bool
	ShowDistance bool
	IsDemo       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Preferences struct {
	InterestedIn  []string
	MinAge        int
	MaxAge        int
	MaxDistanceKm int
}

type Interest struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID       string
	Position int
	Width    int
	Height   int
}

// Card is the public view of a profile shown to other members. It never
// contains e-mail, birthdate or coordinates.
type Card struct {
	UserID     string
	FirstName  string
	Age        int
	Gender     string
	Bio        string
	City       string
	DistanceKm *int
	IsDemo     bool
	Photos     []Photo
	Interests  []Interest
	// SharedInterests counts interests in common with the viewer.
	SharedInterests int
	LastActiveAt    time.Time
}

// --- HTTP payloads ---------------------------------------------------------

type UpsertProfileRequest struct {
	FirstName string `json:"firstName" binding:"required"`
	Birthdate string `json:"birthdate" binding:"required"`
	Gender    string `json:"gender" binding:"required"`
	Bio       string `json:"bio"`
	City      string `json:"city"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type InterestsRequest struct {
	InterestIDs []int `json:"interestIds"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
}

type PrivacyRequest struct {
	IsVisible    *bool `json:"isVisible" binding:"required"`
	ShowDistance *bool `json:"showDistance" binding:"required"`
}

type PhotoResponse struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type PreferencesResponse struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type MyProfileResponse struct {
	UserID       string              `json:"userId"`
	Exists       bool                `json:"exists"`
	IsComplete   bool                `json:"isComplete"`
	Missing      []string            `json:"missing"`
	FirstName    string              `json:"firstName"`
	Birthdate    string              `json:"birthdate"`
	Age          int                 `json:"age"`
	Gender       string              `json:"gender"`
	Bio          string              `json:"bio"`
	City         string              `json:"city"`
	HasLocation  bool                `json:"hasLocation"`
	IsVisible    bool                `json:"isVisible"`
	ShowDistance bool                `json:"showDistance"`
	IsDemo       bool                `json:"isDemo"`
	Photos       []PhotoResponse     `json:"photos"`
	Interests    []Interest          `json:"interests"`
	Preferences  PreferencesResponse `json:"preferences"`
}

type CardResponse struct {
	UserID          string          `json:"userId"`
	FirstName       string          `json:"firstName"`
	Age             int             `json:"age"`
	Gender          string          `json:"gender"`
	Bio             string          `json:"bio"`
	City            string          `json:"city,omitempty"`
	DistanceKm      *int            `json:"distanceKm,omitempty"`
	IsDemo          bool            `json:"isDemo"`
	Photos          []PhotoResponse `json:"photos"`
	Interests       []Interest      `json:"interests"`
	SharedInterests int             `json:"sharedInterests"`
}

type InterestsResponse struct {
	Interests []Interest `json:"interests"`
}
