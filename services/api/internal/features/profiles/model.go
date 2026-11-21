package profiles

import "time"

const (
	GenderWoman     = "woman"
	GenderMan       = "man"
	GenderNonBinary = "nonbinary"

	MinAge          = 18
	MaxAge          = 99
	MaxInterests    = 10
	MaxPhotos       = 6
	MinDistanceKm   = 1
	MaxDistanceKm   = 500
	DefaultDistance = 50
)

var Genders = []string{GenderWoman, GenderMan, GenderNonBinary}

var RelationshipGoals = []string{"long_term", "short_term", "friendship", "undecided"}

// Profile is the private, owner-facing profile.
type Profile struct {
	UserID           string
	FirstName        string
	Birthdate        time.Time
	Gender           string
	Bio              string
	City             string
	JobTitle         string
	HeightCm         *int
	RelationshipGoal *string
	HasLocation      bool
	IsVisible        bool
	ShowDistance     bool
	IsDemo           bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type Preferences struct {
	InterestedIn  []string
	MinAge        int
	MaxAge        int
	MaxDistanceKm int
}

type Interest struct {
	ID    int
	Slug  string
	Label string
}

type Photo struct {
	ID       string
	Position int
	Width    int
	Height   int
}

// ProfileInput is the validated payload for creating/updating a profile.
type ProfileInput struct {
	FirstName        string
	Birthdate        time.Time
	Gender           string
	Bio              string
	City             string
	JobTitle         string
	HeightCm         *int
	RelationshipGoal *string
}

// CardRecord is the storage representation of a public profile.
type CardRecord struct {
	UserID           string
	FirstName        string
	Age              int
	Gender           string
	Bio              string
	City             string
	JobTitle         string
	HeightCm         *int
	RelationshipGoal *string
	DistanceKm       *float64
	IsDemo           bool
	Photos           []Photo
	Interests        []Interest
}

// --- HTTP payloads ---

type InterestResponse struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type InterestsResponse struct {
	Interests []InterestResponse `json:"interests"`
}

type PhotoResponse struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type PreferencesResponse struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type OwnProfileResponse struct {
	FirstName        string  `json:"firstName"`
	Birthdate        string  `json:"birthdate"`
	Age              int     `json:"age"`
	Gender           string  `json:"gender"`
	Bio              string  `json:"bio"`
	City             string  `json:"city"`
	JobTitle         string  `json:"jobTitle"`
	HeightCm         *int    `json:"heightCm"`
	RelationshipGoal *string `json:"relationshipGoal"`
	HasLocation      bool    `json:"hasLocation"`
	IsVisible        bool    `json:"isVisible"`
	ShowDistance     bool    `json:"showDistance"`
}

type MeProfileResponse struct {
	UserID      string               `json:"userId"`
	Email       string               `json:"email"`
	Profile     *OwnProfileResponse  `json:"profile"`
	Preferences *PreferencesResponse `json:"preferences"`
	Photos      []PhotoResponse      `json:"photos"`
	Interests   []InterestResponse   `json:"interests"`
	IsComplete  bool                 `json:"isComplete"`
	Missing     []string             `json:"missing"`
}

// CardResponse is the public view of a member. It never contains the email,
// birthdate or coordinates.
type CardResponse struct {
	UserID           string             `json:"userId"`
	FirstName        string             `json:"firstName"`
	Age              int                `json:"age"`
	Gender           string             `json:"gender"`
	Bio              string             `json:"bio"`
	City             string             `json:"city"`
	JobTitle         string             `json:"jobTitle"`
	HeightCm         *int               `json:"heightCm"`
	RelationshipGoal *string            `json:"relationshipGoal"`
	DistanceKm       *int               `json:"distanceKm"`
	IsDemo           bool               `json:"isDemo"`
	Photos           []PhotoResponse    `json:"photos"`
	Interests        []InterestResponse `json:"interests"`
}

// SummaryResponse is the compact view used in lists (matches, chats, blocks).
type SummaryResponse struct {
	ID        string  `json:"id"`
	FirstName string  `json:"firstName"`
	Age       int     `json:"age"`
	PhotoURL  *string `json:"photoUrl"`
	IsDemo    bool    `json:"isDemo"`
}

type UpsertProfileRequest struct {
	FirstName        string  `json:"firstName" binding:"required,max=80"`
	Birthdate        string  `json:"birthdate" binding:"required,max=10"`
	Gender           string  `json:"gender" binding:"required,max=20"`
	Bio              string  `json:"bio" binding:"max=2000"`
	City             string  `json:"city" binding:"max=200"`
	JobTitle         string  `json:"jobTitle" binding:"max=200"`
	HeightCm         *int    `json:"heightCm"`
	RelationshipGoal *string `json:"relationshipGoal"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,max=3"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type UpdateInterestsRequest struct {
	InterestIDs []int `json:"interestIds" binding:"max=20"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
}

type UpdateSettingsRequest struct {
	IsVisible    *bool `json:"isVisible"`
	ShowDistance *bool `json:"showDistance"`
}
