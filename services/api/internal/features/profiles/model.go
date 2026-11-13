package profiles

import "time"

var (
	Genders           = []string{"woman", "man", "nonbinary"}
	RelationshipGoals = []string{"long_term", "short_term", "friendship", "unsure"}
)

const (
	MinAge            = 18
	MaxAge            = 99
	MaxInterests      = 10
	MaxFirstNameRunes = 40
	MaxBioRunes       = 500
	MaxJobTitleRunes  = 60
	MaxCityRunes      = 80
	MinHeightCm       = 120
	MaxHeightCm       = 230
	MaxDistanceKm     = 500
)

type Interest struct {
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Category string `json:"category,omitempty"`
}

type PhotoView struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// OwnProfile is everything a member sees and edits about themselves.
type OwnProfile struct {
	UserID           string      `json:"userId"`
	FirstName        string      `json:"firstName"`
	Birthdate        *string     `json:"birthdate"`
	Age              *int        `json:"age"`
	Gender           *string     `json:"gender"`
	Bio              string      `json:"bio"`
	JobTitle         string      `json:"jobTitle"`
	HeightCm         *int        `json:"heightCm"`
	RelationshipGoal *string     `json:"relationshipGoal"`
	City             string      `json:"city"`
	HasLocation      bool        `json:"hasLocation"`
	IsPaused         bool        `json:"isPaused"`
	ShowDistance     bool        `json:"showDistance"`
	IsComplete       bool        `json:"isComplete"`
	Missing          []string    `json:"missing"`
	Photos           []PhotoView `json:"photos"`
	Interests        []Interest  `json:"interests"`
	Preferences      Preferences `json:"preferences"`
}

// PublicProfile is what another member may see. It never contains e-mail,
// birthdate or coordinates; distance is approximate and optional.
type PublicProfile struct {
	UserID           string      `json:"userId"`
	FirstName        string      `json:"firstName"`
	Age              int         `json:"age"`
	Gender           string      `json:"gender"`
	Bio              string      `json:"bio"`
	JobTitle         string      `json:"jobTitle"`
	HeightCm         *int        `json:"heightCm"`
	RelationshipGoal *string     `json:"relationshipGoal"`
	City             string      `json:"city"`
	DistanceKm       *int        `json:"distanceKm"`
	Photos           []PhotoView `json:"photos"`
	Interests        []Interest  `json:"interests"`
	SharedInterests  []string    `json:"sharedInterests"`
	IsDemo           bool        `json:"isDemo"`
	LastActiveAt     time.Time   `json:"-"`
}

// PrimaryPhotoURL returns the first photo URL or an empty string.
func (p PublicProfile) PrimaryPhotoURL() string {
	if len(p.Photos) == 0 {
		return ""
	}
	return p.Photos[0].URL
}

// UpdateProfileRequest is a partial update: nil fields are left untouched.
type UpdateProfileRequest struct {
	FirstName        *string   `json:"firstName"`
	Birthdate        *string   `json:"birthdate"`
	Gender           *string   `json:"gender"`
	Bio              *string   `json:"bio"`
	JobTitle         *string   `json:"jobTitle"`
	HeightCm         *int      `json:"heightCm"`
	RelationshipGoal *string   `json:"relationshipGoal"`
	Interests        *[]string `json:"interests"`
	IsPaused         *bool     `json:"isPaused"`
	ShowDistance     *bool     `json:"showDistance"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      string   `json:"city" binding:"max=200"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  *[]string `json:"interestedIn"`
	AgeMin        *int      `json:"ageMin"`
	AgeMax        *int      `json:"ageMax"`
	MaxDistanceKm *int      `json:"maxDistanceKm"`
}

// profilePatch is the validated, normalized form of UpdateProfileRequest.
type profilePatch struct {
	FirstName         *string
	Birthdate         *time.Time
	Gender            *string
	Bio               *string
	JobTitle          *string
	HeightCm          *int
	ClearHeight       bool
	RelationshipGoal  *string
	ClearRelationGoal bool
	InterestSlugs     *[]string
	IsPaused          *bool
	ShowDistance      *bool
}

type storedProfile struct {
	UserID           string
	FirstName        string
	Birthdate        *time.Time
	Gender           *string
	Bio              string
	JobTitle         string
	HeightCm         *int
	RelationshipGoal *string
	City             string
	HasLocation      bool
	IsPaused         bool
	ShowDistance     bool
	IsComplete       bool
	Preferences      Preferences
}

type storedCard struct {
	UserID           string
	FirstName        string
	Birthdate        time.Time
	Gender           string
	Bio              string
	JobTitle         string
	HeightCm         *int
	RelationshipGoal *string
	City             string
	DistanceKm       *float64
	ShowDistance     bool
	IsDemo           bool
	LastActiveAt     time.Time
}

type storedPhoto struct {
	ID       string
	UserID   string
	Position int
	Width    int
	Height   int
}
