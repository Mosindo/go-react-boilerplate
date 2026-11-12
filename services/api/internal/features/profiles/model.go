package profiles

import "time"

const (
	MinAge            = 18
	MaxAge            = 99
	MaxInterests      = 10
	MaxPhotos         = 6
	maxFirstNameRunes = 40
	maxBioRunes       = 500
	maxOccupationRune = 80
	maxCityRunes      = 80
)

var (
	Genders           = []string{"woman", "man", "nonbinary"}
	RelationshipGoals = []string{"long_term", "short_term", "friendship", "unsure"}
)

type Interest struct {
	ID       int    `json:"id"`
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Category string `json:"category"`
}

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Position int    `json:"position"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type Privacy struct {
	Discoverable bool `json:"discoverable"`
	ShowAge      bool `json:"showAge"`
	ShowDistance bool `json:"showDistance"`
}

// OwnProfile is what the owner sees: complete data, never other people's.
type OwnProfile struct {
	UserID            string      `json:"userId"`
	FirstName         string      `json:"firstName"`
	Birthdate         string      `json:"birthdate"`
	Age               int         `json:"age"`
	Gender            string      `json:"gender"`
	Bio               string      `json:"bio"`
	Occupation        string      `json:"occupation"`
	RelationshipGoal  string      `json:"relationshipGoal"`
	City              string      `json:"city"`
	HasLocation       bool        `json:"hasLocation"`
	LocationUpdatedAt *time.Time  `json:"locationUpdatedAt,omitempty"`
	Privacy           Privacy     `json:"privacy"`
	Preferences       Preferences `json:"preferences"`
	Interests         []Interest  `json:"interests"`
	Photos            []Photo     `json:"photos"`
	IsComplete        bool        `json:"isComplete"`
	Missing           []string    `json:"missing"`
	CreatedAt         time.Time   `json:"createdAt"`
	UpdatedAt         time.Time   `json:"updatedAt"`
}

// PublicProfile is what other members see. It never contains email,
// birthdate or coordinates: only an age and an approximate distance.
type PublicProfile struct {
	UserID           string     `json:"userId"`
	FirstName        string     `json:"firstName"`
	Age              *int       `json:"age,omitempty"`
	Gender           string     `json:"gender"`
	Bio              string     `json:"bio"`
	Occupation       string     `json:"occupation"`
	RelationshipGoal string     `json:"relationshipGoal"`
	City             string     `json:"city"`
	DistanceKm       *int       `json:"distanceKm,omitempty"`
	Interests        []Interest `json:"interests"`
	Photos           []Photo    `json:"photos"`
}

type UpsertProfileRequest struct {
	FirstName        string `json:"firstName" binding:"required,max=200"`
	Birthdate        string `json:"birthdate" binding:"required,max=10"`
	Gender           string `json:"gender" binding:"required,max=20"`
	Bio              string `json:"bio" binding:"max=2000"`
	Occupation       string `json:"occupation" binding:"max=400"`
	RelationshipGoal string `json:"relationshipGoal" binding:"max=20"`
	City             string `json:"city" binding:"max=400"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,max=3"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type InterestsRequest struct {
	InterestIDs []int `json:"interestIds" binding:"max=10"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      *string  `json:"city" binding:"omitempty,max=400"`
}

type PrivacyRequest struct {
	Discoverable *bool `json:"discoverable" binding:"required"`
	ShowAge      *bool `json:"showAge" binding:"required"`
	ShowDistance *bool `json:"showDistance" binding:"required"`
}

type InterestsResponse struct {
	Interests []Interest `json:"interests"`
}

// ProfileInput is the validated, normalized version of UpsertProfileRequest.
type ProfileInput struct {
	FirstName        string
	Birthdate        time.Time
	Gender           string
	Bio              string
	Occupation       string
	RelationshipGoal string
	City             string
}

type storedProfile struct {
	UserID            string
	FirstName         string
	Birthdate         time.Time
	Gender            string
	Bio               string
	Occupation        string
	RelationshipGoal  string
	City              string
	Latitude          *float64
	Longitude         *float64
	LocationUpdatedAt *time.Time
	Privacy           Privacy
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type storedPhoto struct {
	UserID     string
	ID         string
	StorageKey string
	Width      int
	Height     int
	Position   int
}

type publicRow struct {
	UserID           string
	FirstName        string
	Birthdate        time.Time
	Gender           string
	Bio              string
	Occupation       string
	RelationshipGoal string
	City             string
	ShowAge          bool
	ShowDistance     bool
	DistanceKm       *float64
}
