package profiles

import "time"

const (
	GenderWoman     = "woman"
	GenderMan       = "man"
	GenderNonBinary = "nonbinary"

	MinAge = 18
	MaxAge = 99

	MaxPhotos = 6
)

var genders = map[string]struct{}{GenderWoman: {}, GenderMan: {}, GenderNonBinary: {}}

var lookingForValues = map[string]struct{}{"relationship": {}, "casual": {}, "friendship": {}, "open": {}}

// Interest is one entry of the selectable catalogue.
type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

// StoredProfile mirrors the profiles table plus the user's interest slugs.
type StoredProfile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	Occupation   string
	LookingFor   string
	City         string
	Latitude     float64
	Longitude    float64
	Discoverable bool
	ShowDistance bool
	Interests    []string
}

type StoredPreferences struct {
	InterestedIn  []string
	MinAge        int
	MaxAge        int
	MaxDistanceKm int
}

// StoredPublic is the viewer-relative projection used for cards and profile pages.
type StoredPublic struct {
	UserID     string
	FirstName  string
	BirthDate  time.Time
	Gender     string
	Bio        string
	Occupation string
	LookingFor string
	City       string
	DistanceKm *float64
	PhotoIDs   []string
	Interests  []string
}

type ProfileInput struct {
	FirstName  string   `json:"firstName"`
	BirthDate  string   `json:"birthDate"`
	Gender     string   `json:"gender"`
	Bio        string   `json:"bio"`
	Occupation string   `json:"occupation"`
	LookingFor string   `json:"lookingFor"`
	City       string   `json:"city"`
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	Interests  []string `json:"interests"`
}

type PreferencesInput struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type PrivacyInput struct {
	Discoverable *bool `json:"discoverable"`
	ShowDistance *bool `json:"showDistance"`
}

type PhotoRef struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// OwnProfile is what a user sees about themselves. Coordinates are already rounded.
type OwnProfile struct {
	FirstName    string   `json:"firstName"`
	BirthDate    string   `json:"birthDate"`
	Age          int      `json:"age"`
	Gender       string   `json:"gender"`
	Bio          string   `json:"bio"`
	Occupation   string   `json:"occupation"`
	LookingFor   string   `json:"lookingFor"`
	City         string   `json:"city"`
	Latitude     float64  `json:"latitude"`
	Longitude    float64  `json:"longitude"`
	Discoverable bool     `json:"discoverable"`
	ShowDistance bool     `json:"showDistance"`
	Interests    []string `json:"interests"`
}

type PreferencesResponse struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// MeProfileResponse also tells the client which onboarding steps remain.
type MeProfileResponse struct {
	Profile     *OwnProfile          `json:"profile"`
	Preferences *PreferencesResponse `json:"preferences"`
	Photos      []PhotoRef           `json:"photos"`
	IsComplete  bool                 `json:"isComplete"`
	Missing     []string             `json:"missing"`
}

// PublicProfile is the only shape in which other users are ever exposed:
// no email, birth date, coordinates or exact distance.
type PublicProfile struct {
	ID         string     `json:"id"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	Occupation string     `json:"occupation"`
	LookingFor string     `json:"lookingFor"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Photos     []PhotoRef `json:"photos"`
	Interests  []Interest `json:"interests"`
}
