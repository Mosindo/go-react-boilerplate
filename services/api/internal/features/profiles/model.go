package profiles

import "time"

var (
	Genders           = []string{"woman", "man", "nonbinary"}
	RelationshipGoals = []string{"long_term", "short_term", "friendship", "unsure"}
)

type Profile struct {
	UserID            string
	FirstName         string
	Birthdate         time.Time
	Gender            string
	Bio               string
	JobTitle          string
	City              string
	HeightCm          *int
	RelationshipGoal  string
	Latitude          *float64
	Longitude         *float64
	LocationUpdatedAt *time.Time
	IsDiscoverable    bool
	ShowDistance      bool
	ModerationHidden  bool
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type Interest struct {
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Category string `json:"category"`
}

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// ProfileView is the owner's view of their profile. Raw coordinates are never
// returned, not even to the owner: only whether a location is set.
type ProfileView struct {
	FirstName         string     `json:"firstName"`
	Birthdate         string     `json:"birthdate"`
	Age               int        `json:"age"`
	Gender            string     `json:"gender"`
	Bio               string     `json:"bio"`
	JobTitle          string     `json:"jobTitle"`
	City              string     `json:"city"`
	HeightCm          *int       `json:"heightCm"`
	RelationshipGoal  string     `json:"relationshipGoal"`
	HasLocation       bool       `json:"hasLocation"`
	LocationUpdatedAt *time.Time `json:"locationUpdatedAt"`
	IsDiscoverable    bool       `json:"isDiscoverable"`
	ShowDistance      bool       `json:"showDistance"`
	UnderReview       bool       `json:"underReview"`
}

type MyProfileResponse struct {
	Profile     *ProfileView `json:"profile"`
	Preferences *Preferences `json:"preferences"`
	Photos      []Photo      `json:"photos"`
	Interests   []Interest   `json:"interests"`
	IsComplete  bool         `json:"isComplete"`
	Missing     []string     `json:"missing"`
}

type UpsertProfileRequest struct {
	FirstName        string `json:"firstName" binding:"required,max=60"`
	Birthdate        string `json:"birthdate" binding:"required,len=10"`
	Gender           string `json:"gender" binding:"required,max=20"`
	Bio              string `json:"bio" binding:"max=2000"`
	JobTitle         string `json:"jobTitle" binding:"max=200"`
	City             string `json:"city" binding:"max=200"`
	HeightCm         *int   `json:"heightCm"`
	RelationshipGoal string `json:"relationshipGoal" binding:"max=20"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,max=3"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type PrivacyRequest struct {
	IsDiscoverable *bool `json:"isDiscoverable"`
	ShowDistance   *bool `json:"showDistance"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      string   `json:"city" binding:"max=200"`
}

type InterestsRequest struct {
	Slugs []string `json:"slugs" binding:"max=20"`
}

type InterestsResponse struct {
	Interests []Interest `json:"interests"`
}
