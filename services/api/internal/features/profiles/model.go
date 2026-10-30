package profiles

import "time"

const (
	GenderWoman     = "woman"
	GenderMan       = "man"
	GenderNonBinary = "non_binary"

	MinAge       = 18
	MaxAge       = 99
	MaxInterests = 10
)

var validGenders = map[string]struct{}{GenderWoman: {}, GenderMan: {}, GenderNonBinary: {}}

type Interest struct {
	ID    int16  `json:"-"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type PhotoRef struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// OwnProfile is what a user sees about themselves. Exact coordinates are never
// returned, not even to their owner.
type OwnProfile struct {
	UserID         string      `json:"userId"`
	DisplayName    string      `json:"displayName"`
	BirthDate      string      `json:"birthDate"`
	Age            int         `json:"age"`
	Gender         string      `json:"gender"`
	Bio            string      `json:"bio"`
	City           string      `json:"city"`
	HasLocation    bool        `json:"hasLocation"`
	IsDiscoverable bool        `json:"isDiscoverable"`
	ShowDistance   bool        `json:"showDistance"`
	Interests      []Interest  `json:"interests"`
	Photos         []PhotoRef  `json:"photos"`
	Preferences    Preferences `json:"preferences"`
	IsComplete     bool        `json:"isComplete"`
	Missing        []string    `json:"missing"`
}

// PublicProfile is what other users may see.
type PublicProfile struct {
	UserID      string     `json:"userId"`
	DisplayName string     `json:"displayName"`
	Age         int        `json:"age"`
	Gender      string     `json:"gender"`
	Bio         string     `json:"bio"`
	City        string     `json:"city"`
	DistanceKm  *int       `json:"distanceKm,omitempty"`
	Interests   []Interest `json:"interests"`
	Photos      []PhotoRef `json:"photos"`
}

type SaveProfileRequest struct {
	DisplayName    string   `json:"displayName" binding:"max=200"`
	BirthDate      string   `json:"birthDate" binding:"max=20"`
	Gender         string   `json:"gender" binding:"max=20"`
	Bio            string   `json:"bio" binding:"max=2000"`
	City           string   `json:"city" binding:"max=400"`
	Latitude       *float64 `json:"latitude"`
	Longitude      *float64 `json:"longitude"`
	Interests      []string `json:"interests" binding:"max=50"`
	IsDiscoverable *bool    `json:"isDiscoverable"`
	ShowDistance   *bool    `json:"showDistance"`
}

type SavePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"max=10"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// ProfileInput is a validated profile write.
type ProfileInput struct {
	DisplayName    string
	BirthDate      time.Time
	Gender         string
	Bio            string
	City           string
	Latitude       *float64
	Longitude      *float64
	InterestSlugs  []string
	IsDiscoverable *bool
	ShowDistance   *bool
}
