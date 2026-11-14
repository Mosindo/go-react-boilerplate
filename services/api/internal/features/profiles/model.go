package profiles

import "time"

var (
	Genders     = []string{"woman", "man", "nonbinary"}
	LookingFors = []string{"relationship", "casual", "friendship", "not_sure"}
)

const (
	MinAge             = 18
	MaxAge             = 99
	MaxInterests       = 10
	MaxDistanceKm      = 500
	DefaultDistanceKm  = 50
	locationPrecision  = 100.0 // 2 decimals ≈ 1.1 km
	maxFirstNameRunes  = 40
	maxBioRunes        = 500
	maxJobTitleRunes   = 80
	maxCityRunes       = 80
	minHeightCm        = 100
	maxHeightCm        = 250
	birthdateLayout    = "2006-01-02"
	maxProfilesPerCall = 100
)

type Interest struct {
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Category string `json:"category"`
}

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

// Profile is the owner's full view of their own profile.
type Profile struct {
	FirstName    string     `json:"firstName"`
	Birthdate    string     `json:"birthdate"`
	Age          int        `json:"age"`
	Gender       string     `json:"gender"`
	Bio          string     `json:"bio"`
	JobTitle     string     `json:"jobTitle"`
	HeightCm     *int       `json:"heightCm"`
	LookingFor   string     `json:"lookingFor"`
	City         string     `json:"city"`
	HasLocation  bool       `json:"hasLocation"`
	Discoverable bool       `json:"discoverable"`
	ShowDistance bool       `json:"showDistance"`
	Interests    []Interest `json:"interests"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// MyProfileResponse is returned by GET /me/profile.
type MyProfileResponse struct {
	Profile     *Profile     `json:"profile"`
	Preferences *Preferences `json:"preferences"`
	Photos      []Photo      `json:"photos"`
	Complete    bool         `json:"complete"`
}

// PublicProfile is what other members can see. It never contains e-mail,
// birthdate or coordinates: only age and an approximate distance.
type PublicProfile struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	JobTitle   string     `json:"jobTitle"`
	HeightCm   *int       `json:"heightCm"`
	LookingFor string     `json:"lookingFor"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Photos     []Photo    `json:"photos"`
	Interests  []Interest `json:"interests"`
}

type UpdateProfileRequest struct {
	FirstName  string   `json:"firstName"`
	Birthdate  string   `json:"birthdate"`
	Gender     string   `json:"gender"`
	Bio        string   `json:"bio"`
	JobTitle   string   `json:"jobTitle"`
	HeightCm   *int     `json:"heightCm"`
	LookingFor string   `json:"lookingFor"`
	Interests  []string `json:"interests"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	City      string   `json:"city"`
}

type UpdatePrivacyRequest struct {
	Discoverable *bool `json:"discoverable"`
	ShowDistance *bool `json:"showDistance"`
}

// ProfileInput is a validated, normalised UpdateProfileRequest.
type ProfileInput struct {
	FirstName  string
	Birthdate  time.Time
	Gender     string
	Bio        string
	JobTitle   string
	HeightCm   *int
	LookingFor string
	Interests  []string
}
