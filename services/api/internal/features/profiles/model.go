package profiles

import "time"

type Gender string

const (
	GenderWoman     Gender = "woman"
	GenderMan       Gender = "man"
	GenderNonBinary Gender = "non_binary"
)

var AllGenders = []Gender{GenderWoman, GenderMan, GenderNonBinary}

func (g Gender) valid() bool {
	for _, x := range AllGenders {
		if x == g {
			return true
		}
	}
	return false
}

const (
	MaxFirstName     = 40
	MaxBio           = 500
	MaxCity          = 80
	MaxInterests     = 10
	maxRawInterests  = 50
	MinPrefAge       = 18
	MaxPrefAge       = 99
	MinPrefDistance  = 1
	MaxPrefDistance  = 500
	defaultDistance  = 50
	defaultAgeMin    = 18
	defaultAgeMax    = 99
	locationDecimals = 100.0
)

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

// PublicProfile is what other users see.
type PublicProfile struct {
	UserID     string   `json:"userId"`
	FirstName  string   `json:"firstName"`
	Age        *int     `json:"age"`
	Gender     Gender   `json:"gender"`
	Bio        string   `json:"bio"`
	City       string   `json:"city"`
	DistanceKm *int     `json:"distanceKm"`
	Interests  []string `json:"interests"`
	Photos     []Photo  `json:"photos"`
}

// MyProfile is the caller's own profile: PublicProfile plus settings.
type MyProfile struct {
	PublicProfile
	HasLocation  bool `json:"hasLocation"`
	ShowDistance bool `json:"showDistance"`
	ShowAge      bool `json:"showAge"`
	Discoverable bool `json:"discoverable"`
	IsComplete   bool `json:"isComplete"`
}

type Preferences struct {
	InterestedIn  []Gender `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type ProfileRequest struct {
	FirstName    string   `json:"firstName"`
	Gender       string   `json:"gender"`
	Bio          string   `json:"bio"`
	City         string   `json:"city"`
	Interests    []string `json:"interests"`
	ShowDistance *bool    `json:"showDistance"`
	ShowAge      *bool    `json:"showAge"`
	Discoverable *bool    `json:"discoverable"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        *int     `json:"ageMin"`
	AgeMax        *int     `json:"ageMax"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
}

// ProfileInput is a validated, normalised profile write.
type ProfileInput struct {
	FirstName    string
	Gender       Gender
	Bio          string
	City         string
	Interests    []string // deduplicated slugs
	ShowDistance *bool
	ShowAge      *bool
	Discoverable *bool
}

// storedProfile is the raw row for a profile before photo URLs are minted.
type storedProfile struct {
	UserID       string
	FirstName    string
	Gender       Gender
	Bio          string
	City         string
	Age          *int
	ShowAge      bool
	ShowDistance bool
	Discoverable bool
	Lat, Lng     *float64
	ViewerLat    *float64
	ViewerLng    *float64
	Matched      bool
	Blocked      bool
	HasPhoto     bool
	Interests    []string
	Photos       []storedPhoto
	UpdatedAt    time.Time
}

type storedPhoto struct {
	ID       string
	Position int
}
