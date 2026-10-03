package profiles

const (
	GenderMan       = "man"
	GenderWoman     = "woman"
	GenderNonBinary = "nonbinary"

	MinAge       = 18
	MaxAge       = 99
	MaxInterests = 10
)

var validGenders = map[string]struct{}{GenderMan: {}, GenderWoman: {}, GenderNonBinary: {}}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

// PublicProfile is what other users are allowed to see. It never carries the birth date,
// the email or any coordinates.
type PublicProfile struct {
	ID         string     `json:"id"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
	DistanceKm *int       `json:"distanceKm,omitempty"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
}

// OwnProfile is the full view of the caller's own profile.
type OwnProfile struct {
	PublicProfile
	BirthDate    string      `json:"birthDate"`
	IsVisible    bool        `json:"isVisible"`
	ShowDistance bool        `json:"showDistance"`
	HasLocation  bool        `json:"hasLocation"`
	Preferences  Preferences `json:"preferences"`
	// Discoverable is true once the profile has at least one photo and is visible.
	Discoverable bool `json:"discoverable"`
}

type UpsertProfileRequest struct {
	FirstName string   `json:"firstName" binding:"required,max=200"`
	BirthDate string   `json:"birthDate" binding:"required,max=10"`
	Gender    string   `json:"gender" binding:"required,max=20"`
	Bio       string   `json:"bio" binding:"max=2000"`
	City      string   `json:"city" binding:"max=200"`
	Interests []string `json:"interests" binding:"max=50,dive,max=64"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type LocationRequest struct {
	Latitude  float64 `json:"latitude" binding:"required"`
	Longitude float64 `json:"longitude" binding:"required"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,min=1,max=3,dive,max=20"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
}

type PrivacyRequest struct {
	IsVisible    *bool `json:"isVisible" binding:"required"`
	ShowDistance *bool `json:"showDistance" binding:"required"`
}
