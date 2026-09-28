package profiles

import "time"

const (
	GenderMan       = "man"
	GenderWoman     = "woman"
	GenderNonBinary = "non_binary"

	MaxInterests = 10
	MaxBio       = 500
	MaxFirstName = 50
	MaxLabel     = 80
)

var Genders = []string{GenderMan, GenderWoman, GenderNonBinary}

func ValidGender(g string) bool {
	for _, v := range Genders {
		if v == g {
			return true
		}
	}
	return false
}

type Interest struct {
	ID    int16  `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

// PhotoURL is the authenticated content path of a photo.
func PhotoURL(id string) string { return "/photos/" + id + "/content" }

// Profile is the caller's own profile (includes birthDate).
type Profile struct {
	UserID         string     `json:"userId"`
	FirstName      string     `json:"firstName"`
	Age            int        `json:"age"`
	BirthDate      string     `json:"birthDate"`
	Gender         string     `json:"gender"`
	Bio            string     `json:"bio"`
	LocationLabel  string     `json:"locationLabel"`
	HasLocation    bool       `json:"hasLocation"`
	ShowDistance   bool       `json:"showDistance"`
	IsDiscoverable bool       `json:"isDiscoverable"`
	Interests      []Interest `json:"interests"`
	Photos         []Photo    `json:"photos"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type Me struct {
	ID              string      `json:"id"`
	Email           string      `json:"email"`
	CreatedAt       time.Time   `json:"createdAt"`
	Profile         *Profile    `json:"profile"`
	Preferences     Preferences `json:"preferences"`
	ProfileComplete bool        `json:"profileComplete"`
}

type UpdateProfileRequest struct {
	FirstName      string  `json:"firstName"`
	BirthDate      string  `json:"birthDate"`
	Gender         string  `json:"gender"`
	Bio            string  `json:"bio"`
	InterestIDs    []int16 `json:"interestIds"`
	ShowDistance   *bool   `json:"showDistance"`
	IsDiscoverable *bool   `json:"isDiscoverable"`
}

type UpdateLocationRequest struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Label     *string  `json:"label"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// ProfileRecord is the stored row (no photos/interests).
type ProfileRecord struct {
	UserID         string
	FirstName      string
	BirthDate      time.Time
	Gender         string
	Bio            string
	LocationLabel  string
	HasLocation    bool
	ShowDistance   bool
	IsDiscoverable bool
}

type ProfileInput struct {
	FirstName      string
	BirthDate      time.Time
	Gender         string
	Bio            string
	InterestIDs    []int16
	ShowDistance   *bool
	IsDiscoverable *bool
}
