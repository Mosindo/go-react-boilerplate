package profiles

import "time"

var (
	Genders            = []string{"woman", "man", "nonbinary"}
	RelationshipGoals  = []string{"long_term", "short_term", "friendship", "unsure"}
	MinAge             = 18
	MaxAge             = 99
	MaxInterests       = 10
	MaxDistanceKmLimit = 500
)

type Photo struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Interest struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type Completeness struct {
	Complete bool     `json:"complete"`
	Missing  []string `json:"missing"`
}

// OwnProfile is the full profile returned to its owner only.
type OwnProfile struct {
	UserID           string       `json:"userId"`
	FirstName        string       `json:"firstName"`
	Birthdate        *string      `json:"birthdate"`
	Age              *int         `json:"age"`
	Gender           *string      `json:"gender"`
	Bio              string       `json:"bio"`
	JobTitle         string       `json:"jobTitle"`
	RelationshipGoal *string      `json:"relationshipGoal"`
	City             string       `json:"city"`
	HasLocation      bool         `json:"hasLocation"`
	Discoverable     bool         `json:"discoverable"`
	ShowDistance     bool         `json:"showDistance"`
	Photos           []Photo      `json:"photos"`
	Interests        []Interest   `json:"interests"`
	Preferences      Preferences  `json:"preferences"`
	Completeness     Completeness `json:"completeness"`
}

// PublicProfile is what other members may see. It never contains the email,
// the exact birthdate or coordinates.
type PublicProfile struct {
	UserID           string     `json:"userId"`
	FirstName        string     `json:"firstName"`
	Age              int        `json:"age"`
	Gender           string     `json:"gender"`
	Bio              string     `json:"bio"`
	JobTitle         string     `json:"jobTitle"`
	RelationshipGoal *string    `json:"relationshipGoal"`
	City             string     `json:"city"`
	DistanceKm       *int       `json:"distanceKm"`
	Photos           []Photo    `json:"photos"`
	Interests        []Interest `json:"interests"`
	SharedInterests  int        `json:"sharedInterests"`
}

// Summary is the lightweight identity used in match and conversation lists.
type Summary struct {
	UserID    string `json:"userId"`
	FirstName string `json:"firstName"`
	Age       *int   `json:"age"`
	Photo     *Photo `json:"photo"`
}

type UpdateProfileRequest struct {
	FirstName        *string `json:"firstName"`
	Birthdate        *string `json:"birthdate"`
	Gender           *string `json:"gender"`
	Bio              *string `json:"bio"`
	JobTitle         *string `json:"jobTitle"`
	RelationshipGoal *string `json:"relationshipGoal"`
	City             *string `json:"city"`
	Discoverable     *bool   `json:"discoverable"`
	ShowDistance     *bool   `json:"showDistance"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,max=3"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

type UpdateInterestsRequest struct {
	InterestIDs []int `json:"interestIds" binding:"max=10"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      *string  `json:"city"`
}

// storedProfile mirrors the profiles + preferences rows.
type storedProfile struct {
	UserID           string
	FirstName        string
	Birthdate        *time.Time
	Gender           *string
	Bio              string
	JobTitle         string
	RelationshipGoal *string
	City             string
	HasLocation      bool
	Discoverable     bool
	ShowDistance     bool
	Preferences      Preferences
}

// cardRow is a candidate profile row with the distance to the viewer.
type cardRow struct {
	UserID           string
	FirstName        string
	Birthdate        *time.Time
	Gender           *string
	Bio              string
	JobTitle         string
	RelationshipGoal *string
	City             string
	ShowDistance     bool
	DistanceKm       *float64
}
