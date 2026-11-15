package profiles

import "time"

const (
	MinAge          = 18
	MaxAge          = 99
	MaxInterests    = 10
	MaxBioRunes     = 500
	MaxNameRunes    = 40
	MaxJobRunes     = 60
	MaxCityRunes    = 80
	MinHeightCm     = 120
	MaxHeightCm     = 230
	MaxDistanceKm   = 500
	LocationDecimal = 100 // coordinates are rounded to 1/100 degree (~1.1 km)
)

var Genders = []string{"woman", "man", "nonbinary"}

type Interest struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID         string
	StorageKey string
	Width      int
	Height     int
	Position   int
}

type PhotoResponse struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Position int    `json:"position"`
}

// OwnProfile is the full private view of the current user's profile.
type OwnProfile struct {
	UserID       string
	FirstName    string
	Birthdate    *time.Time
	Gender       *string
	Bio          string
	JobTitle     string
	HeightCm     *int
	City         string
	HasLocation  bool
	IsPaused     bool
	HideDistance bool
	Interests    []Interest
	Photos       []Photo
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type OwnProfileResponse struct {
	UserID       string          `json:"userId"`
	FirstName    string          `json:"firstName"`
	Birthdate    *string         `json:"birthdate"`
	Age          *int            `json:"age"`
	Gender       *string         `json:"gender"`
	Bio          string          `json:"bio"`
	JobTitle     string          `json:"jobTitle"`
	HeightCm     *int            `json:"heightCm"`
	City         string          `json:"city"`
	HasLocation  bool            `json:"hasLocation"`
	IsPaused     bool            `json:"isPaused"`
	HideDistance bool            `json:"hideDistance"`
	Interests    []Interest      `json:"interests"`
	Photos       []PhotoResponse `json:"photos"`
}

type Completeness struct {
	IsComplete bool     `json:"isComplete"`
	Missing    []string `json:"missing"`
}

type MeProfileResponse struct {
	Profile      OwnProfileResponse `json:"profile"`
	Preferences  Preferences        `json:"preferences"`
	Completeness Completeness       `json:"completeness"`
}

// Card is the public view of another user, as shown in discovery, likes,
// matches and conversations. It never contains e-mail, birthdate or
// coordinates.
type Card struct {
	UserID     string          `json:"userId"`
	FirstName  string          `json:"firstName"`
	Age        int             `json:"age"`
	Gender     string          `json:"gender"`
	Bio        string          `json:"bio"`
	JobTitle   string          `json:"jobTitle"`
	HeightCm   *int            `json:"heightCm"`
	City       string          `json:"city"`
	DistanceKm *int            `json:"distanceKm"`
	IsDemo     bool            `json:"isDemo"`
	Interests  []Interest      `json:"interests"`
	Photos     []PhotoResponse `json:"photos"`
}

type UpdateProfileRequest struct {
	FirstName   string `json:"firstName" binding:"max=200"`
	Birthdate   string `json:"birthdate" binding:"max=10"`
	Gender      string `json:"gender" binding:"max=20"`
	Bio         string `json:"bio" binding:"max=2000"`
	JobTitle    string `json:"jobTitle" binding:"max=400"`
	HeightCm    *int   `json:"heightCm"`
	City        string `json:"city" binding:"max=400"`
	InterestIDs []int  `json:"interestIds" binding:"max=50"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      string   `json:"city" binding:"max=400"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,min=1,max=3"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type UpdatePrivacyRequest struct {
	IsPaused     *bool `json:"isPaused" binding:"required"`
	HideDistance *bool `json:"hideDistance" binding:"required"`
}

// ProfileUpdate is the validated, normalised form of UpdateProfileRequest.
type ProfileUpdate struct {
	FirstName   string
	Birthdate   *time.Time
	Gender      *string
	Bio         string
	JobTitle    string
	HeightCm    *int
	City        string
	InterestIDs []int
}
