package profiles

import "time"

var (
	Genders           = []string{"woman", "man", "nonbinary"}
	RelationshipGoals = []string{"long_term", "short_term", "friendship", "undecided"}
)

const (
	MinAge          = 18
	MaxAge          = 99
	MaxInterests    = 8
	MaxBioRunes     = 500
	MaxNameRunes    = 30
	MaxJobRunes     = 60
	MaxCityRunes    = 60
	MinDistanceKm   = 1
	MaxDistanceKm   = 500
	DefaultDistance = 50
)

type Profile struct {
	UserID           string
	FirstName        string
	Birthdate        time.Time
	Gender           string
	Bio              string
	JobTitle         string
	City             string
	HeightCm         *int
	RelationshipGoal *string
	Latitude         *float64
	Longitude        *float64
	IsDiscoverable   bool
	ShowDistance     bool
	ModerationStatus string
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
	ID         string
	Position   int
	StorageKey string
	Width      int
	Height     int
}

// ProfileInput is the validated payload of PUT /profile.
type ProfileInput struct {
	FirstName        string
	Birthdate        time.Time
	Gender           string
	Bio              string
	JobTitle         string
	City             string
	HeightCm         *int
	RelationshipGoal *string
	InterestIDs      []int
}

// Card is the public view of someone else's profile.
type Card struct {
	UserID           string
	FirstName        string
	Age              int
	Gender           string
	Bio              string
	JobTitle         string
	City             string
	HeightCm         *int
	RelationshipGoal *string
	DistanceKm       *int
	Photos           []Photo
	Interests        []Interest
}

// ---------------------------------------------------------------------------
// HTTP payloads
// ---------------------------------------------------------------------------

type UpsertProfileRequest struct {
	FirstName        string  `json:"firstName" binding:"required,max=60"`
	Birthdate        string  `json:"birthdate" binding:"required,datetime=2006-01-02"`
	Gender           string  `json:"gender" binding:"required"`
	Bio              string  `json:"bio" binding:"max=2000"`
	JobTitle         string  `json:"jobTitle" binding:"max=200"`
	City             string  `json:"city" binding:"max=200"`
	HeightCm         *int    `json:"heightCm"`
	RelationshipGoal *string `json:"relationshipGoal"`
	InterestIDs      []int   `json:"interestIds" binding:"max=20"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      *string  `json:"city" binding:"omitempty,max=200"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,min=1,max=3"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type PrivacyRequest struct {
	IsDiscoverable *bool `json:"isDiscoverable" binding:"required"`
	ShowDistance   *bool `json:"showDistance" binding:"required"`
}

type PhotoResponse struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type OwnProfileResponse struct {
	FirstName        string  `json:"firstName"`
	Birthdate        string  `json:"birthdate"`
	Age              int     `json:"age"`
	Gender           string  `json:"gender"`
	Bio              string  `json:"bio"`
	JobTitle         string  `json:"jobTitle"`
	City             string  `json:"city"`
	HeightCm         *int    `json:"heightCm"`
	RelationshipGoal *string `json:"relationshipGoal"`
	HasLocation      bool    `json:"hasLocation"`
	IsDiscoverable   bool    `json:"isDiscoverable"`
	ShowDistance     bool    `json:"showDistance"`
	ModerationStatus string  `json:"moderationStatus"`
}

type PreferencesResponse struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type CompletionResponse struct {
	Complete bool     `json:"complete"`
	Missing  []string `json:"missing"`
}

type MeProfileResponse struct {
	Profile     *OwnProfileResponse  `json:"profile"`
	Preferences *PreferencesResponse `json:"preferences"`
	Photos      []PhotoResponse      `json:"photos"`
	Interests   []Interest           `json:"interests"`
	Completion  CompletionResponse   `json:"completion"`
}

type CardResponse struct {
	UserID           string          `json:"userId"`
	FirstName        string          `json:"firstName"`
	Age              int             `json:"age"`
	Gender           string          `json:"gender"`
	Bio              string          `json:"bio"`
	JobTitle         string          `json:"jobTitle"`
	City             string          `json:"city"`
	HeightCm         *int            `json:"heightCm"`
	RelationshipGoal *string         `json:"relationshipGoal"`
	DistanceKm       *int            `json:"distanceKm"`
	Photos           []PhotoResponse `json:"photos"`
	Interests        []Interest      `json:"interests"`
}
