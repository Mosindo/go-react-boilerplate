package profiles

import "time"

const (
	GenderMan       = "man"
	GenderWoman     = "woman"
	GenderNonBinary = "non_binary"

	MinAge          = 18
	MaxAge          = 99
	MaxPhotos       = 6
	MaxInterests    = 10
	MaxBioRunes     = 500
	MaxFirstName    = 40
	MaxCityRunes    = 80
	DefaultMaxDist  = 50
	MaxDistanceKM   = 20000
	distanceBucket  = 5
	coordPrecisionF = 100 // coordinates are rounded to 0.01 degree (~1 km)
)

var validGenders = map[string]struct{}{GenderMan: {}, GenderWoman: {}, GenderNonBinary: {}}

// Photo is a picture reference. Bytes are only served by an authorized endpoint.
type Photo struct {
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
	MaxDistanceKM int      `json:"maxDistanceKm"`
}

type Interest struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

// Profile is the private view returned to its owner.
type Profile struct {
	UserID       string      `json:"userId"`
	FirstName    string      `json:"firstName"`
	BirthDate    string      `json:"birthDate"`
	Age          int         `json:"age"`
	Gender       string      `json:"gender"`
	Bio          string      `json:"bio"`
	City         string      `json:"city"`
	HasLocation  bool        `json:"hasLocation"`
	Discoverable bool        `json:"discoverable"`
	ShowDistance bool        `json:"showDistance"`
	Interests    []string    `json:"interests"`
	Photos       []Photo     `json:"photos"`
	Preferences  Preferences `json:"preferences"`
	Complete     bool        `json:"complete"`
	CreatedAt    time.Time   `json:"createdAt"`
}

// PublicProfile is what other users may see: no birth date, no coordinates,
// only an approximate distance when the owner allows it.
type PublicProfile struct {
	UserID     string   `json:"userId"`
	FirstName  string   `json:"firstName"`
	Age        int      `json:"age"`
	Gender     string   `json:"gender"`
	Bio        string   `json:"bio"`
	City       string   `json:"city"`
	DistanceKM *int     `json:"distanceKm"`
	Interests  []string `json:"interests"`
	Photos     []Photo  `json:"photos"`
}

type UpsertProfileRequest struct {
	FirstName string   `json:"firstName"`
	BirthDate string   `json:"birthDate"`
	Gender    string   `json:"gender"`
	Bio       string   `json:"bio"`
	City      string   `json:"city"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Interests []string `json:"interests"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKM int      `json:"maxDistanceKm"`
}

type UpdatePrivacyRequest struct {
	Discoverable *bool `json:"discoverable"`
	ShowDistance *bool `json:"showDistance"`
}

type ReorderPhotosRequest struct {
	IDs []string `json:"ids"`
}

// ProfileInput is the validated, normalized form handed to the repository.
type ProfileInput struct {
	FirstName string
	BirthDate time.Time
	Gender    string
	Bio       string
	City      string
	Latitude  *float64
	Longitude *float64
	Interests []string
}
