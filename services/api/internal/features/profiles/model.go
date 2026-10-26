package profiles

import "time"

const (
	GenderMan       = "man"
	GenderWoman     = "woman"
	GenderNonBinary = "nonbinary"

	MinAge       = 18
	MaxAge       = 99
	MaxInterests = 10
	MaxPhotos    = 6
)

var validGenders = map[string]struct{}{GenderMan: {}, GenderWoman: {}, GenderNonBinary: {}}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// PublicProfile is what other users may see. It never contains birth date,
// email or coordinates.
type PublicProfile struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	JobTitle   string     `json:"jobTitle"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// OwnProfile is the private view returned to the account owner.
type OwnProfile struct {
	PublicProfile
	BirthDate    string       `json:"birthDate"`
	HasLocation  bool         `json:"hasLocation"`
	IsVisible    bool         `json:"isVisible"`
	ShowDistance bool         `json:"showDistance"`
	Preferences  *Preferences `json:"preferences"`
	IsComplete   bool         `json:"isComplete"`
	Missing      []string     `json:"missing"`
	CreatedAt    time.Time    `json:"createdAt"`
}

type SaveProfileRequest struct {
	FirstName string   `json:"firstName"`
	BirthDate string   `json:"birthDate"`
	Gender    string   `json:"gender"`
	Bio       string   `json:"bio"`
	JobTitle  string   `json:"jobTitle"`
	Interests []string `json:"interests"`
}

type SavePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type SaveLocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      string  `json:"city"`
}

type SavePrivacyRequest struct {
	IsVisible    bool `json:"isVisible"`
	ShowDistance bool `json:"showDistance"`
}
