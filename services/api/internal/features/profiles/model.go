package profiles

import "time"

const (
	MaxInterests = 10
	MinAge       = 18
	MaxAge       = 99
)

var (
	validGenders    = map[string]bool{"man": true, "woman": true, "non_binary": true}
	validLookingFor = map[string]bool{"relationship": true, "casual": true, "friends": true, "open": true}
)

type UpdateProfileRequest struct {
	FirstName  string   `json:"firstName" binding:"required,max=100"`
	Birthdate  string   `json:"birthdate" binding:"required,max=10"`
	Gender     string   `json:"gender" binding:"required,max=20"`
	Bio        string   `json:"bio" binding:"max=2000"`
	Occupation string   `json:"occupation" binding:"max=400"`
	LookingFor string   `json:"lookingFor" binding:"max=20"`
	City       string   `json:"city" binding:"max=400"`
	Interests  []string `json:"interests" binding:"max=50,dive,max=40"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,max=3,dive,max=20"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
	IsVisible     bool     `json:"isVisible"`
	ShowDistance  bool     `json:"showDistance"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
	City      string   `json:"city" binding:"max=400"`
}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
	IsVisible     bool     `json:"isVisible"`
	ShowDistance  bool     `json:"showDistance"`
}

// OwnProfile is the private view of one's own profile.
type OwnProfile struct {
	FirstName   string     `json:"firstName"`
	Birthdate   string     `json:"birthdate"`
	Age         int        `json:"age"`
	Gender      string     `json:"gender"`
	Bio         string     `json:"bio"`
	Occupation  string     `json:"occupation"`
	LookingFor  string     `json:"lookingFor"`
	City        string     `json:"city"`
	HasLocation bool       `json:"hasLocation"`
	Interests   []Interest `json:"interests"`
}

type Onboarding struct {
	HasProfile bool `json:"hasProfile"`
	HasPhoto   bool `json:"hasPhoto"`
	Complete   bool `json:"complete"`
}

type AccountInfo struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type MeResponse struct {
	User        AccountInfo `json:"user"`
	Profile     *OwnProfile `json:"profile"`
	Preferences Preferences `json:"preferences"`
	Photos      []Photo     `json:"photos"`
	Onboarding  Onboarding  `json:"onboarding"`
}

// PublicProfile is what other users see. It never carries exact coordinates or birthdate.
type PublicProfile struct {
	ID              string     `json:"id"`
	FirstName       string     `json:"firstName"`
	Age             int        `json:"age"`
	Gender          string     `json:"gender"`
	Bio             string     `json:"bio"`
	Occupation      string     `json:"occupation"`
	LookingFor      string     `json:"lookingFor"`
	City            string     `json:"city"`
	DistanceKm      *int       `json:"distanceKm"`
	Interests       []Interest `json:"interests"`
	SharedInterests int        `json:"sharedInterests"`
	Photos          []Photo    `json:"photos"`
}
