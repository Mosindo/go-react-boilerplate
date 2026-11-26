package profiles

import "time"

const (
	GenderWoman     = "woman"
	GenderMan       = "man"
	GenderNonBinary = "nonbinary"

	MinAge           = 18
	MaxAge           = 99
	MaxInterests     = 10
	MaxDistanceKm    = 500
	maxFirstNameLen  = 40
	maxBioLen        = 500
	maxJobTitleLen   = 60
	maxCityLen       = 60
	birthdateLayout  = "2006-01-02"
	defaultMaxDistKm = 50
)

var validGenders = map[string]struct{}{GenderWoman: {}, GenderMan: {}, GenderNonBinary: {}}

type Interest struct {
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Category string `json:"category"`
}

type PhotoView struct {
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

// Completion tells the client which onboarding steps remain. A profile is only
// discoverable (and can only swipe) once Complete is true.
type Completion struct {
	Profile     bool `json:"profile"`
	Preferences bool `json:"preferences"`
	Photos      bool `json:"photos"`
	Complete    bool `json:"complete"`
}

type MyProfile struct {
	UserID       string       `json:"userId"`
	FirstName    string       `json:"firstName"`
	Birthdate    string       `json:"birthdate"`
	Age          int          `json:"age"`
	Gender       string       `json:"gender"`
	Bio          string       `json:"bio"`
	JobTitle     string       `json:"jobTitle"`
	City         string       `json:"city"`
	HasLocation  bool         `json:"hasLocation"`
	Discoverable bool         `json:"discoverable"`
	ShowDistance bool         `json:"showDistance"`
	Interests    []Interest   `json:"interests"`
	Photos       []PhotoView  `json:"photos"`
	Preferences  *Preferences `json:"preferences"`
	Completion   Completion   `json:"completion"`
}

// PublicProfile is what other members see. It never contains coordinates, e-mail or birthdate.
type PublicProfile struct {
	UserID     string      `json:"userId"`
	FirstName  string      `json:"firstName"`
	Age        int         `json:"age"`
	Gender     string      `json:"gender"`
	Bio        string      `json:"bio"`
	JobTitle   string      `json:"jobTitle"`
	City       string      `json:"city"`
	DistanceKm *int        `json:"distanceKm"`
	Interests  []Interest  `json:"interests"`
	Photos     []PhotoView `json:"photos"`
	IsDemo     bool        `json:"isDemo"`
}

type UpdateProfileRequest struct {
	FirstName string   `json:"firstName"`
	Birthdate string   `json:"birthdate"`
	Gender    string   `json:"gender"`
	Bio       string   `json:"bio"`
	JobTitle  string   `json:"jobTitle"`
	City      string   `json:"city"`
	Interests []string `json:"interests"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
}

type UpdatePrivacyRequest struct {
	Discoverable *bool `json:"discoverable"`
	ShowDistance *bool `json:"showDistance"`
}

type storedProfile struct {
	UserID       string
	FirstName    string
	Birthdate    time.Time
	Gender       string
	Bio          string
	JobTitle     string
	City         string
	Latitude     *float64
	Longitude    *float64
	Discoverable bool
	ShowDistance bool
	IsDemo       bool
}

type storedPhoto struct {
	ID         string
	UserID     string
	StorageKey string
	Width      int
	Height     int
	Position   int
}
