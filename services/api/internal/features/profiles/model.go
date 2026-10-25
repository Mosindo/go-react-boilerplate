package profiles

import "time"

type Gender string

const (
	GenderMan       Gender = "man"
	GenderWoman     Gender = "woman"
	GenderNonBinary Gender = "nonbinary"
)

func validGender(g Gender) bool {
	return g == GenderMan || g == GenderWoman || g == GenderNonBinary
}

const (
	MinAge        = 18
	MaxAge        = 99
	MaxInterests  = 10
	MaxFirstName  = 40
	MaxBio        = 500
	MaxCity       = 80
	MaxDistanceKm = 500
)

type Photo struct {
	ID        string `json:"id"`
	Position  int    `json:"position"`
	IsPrimary bool   `json:"isPrimary"`
	URL       string `json:"url"`
	ThumbURL  string `json:"thumbUrl"`
}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type MyProfile struct {
	UserID       string     `json:"userId"`
	FirstName    string     `json:"firstName"`
	BirthDate    string     `json:"birthDate"`
	Age          int        `json:"age"`
	Gender       Gender     `json:"gender"`
	Bio          string     `json:"bio"`
	City         string     `json:"city"`
	HasLocation  bool       `json:"hasLocation"`
	Discoverable bool       `json:"discoverable"`
	ShowDistance bool       `json:"showDistance"`
	Interests    []Interest `json:"interests"`
	Photos       []Photo    `json:"photos"`
}

type Preferences struct {
	InterestedIn  []Gender `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
}

type PublicProfile struct {
	UserID          string     `json:"userId"`
	FirstName       string     `json:"firstName"`
	Age             int        `json:"age"`
	Bio             string     `json:"bio"`
	City            string     `json:"city"`
	DistanceKm      *int       `json:"distanceKm"`
	Interests       []Interest `json:"interests"`
	Photos          []Photo    `json:"photos"`
	SharedInterests []string   `json:"sharedInterests"`
}

type ProfileState struct {
	Profile     *MyProfile   `json:"profile"`
	Preferences *Preferences `json:"preferences"`
	Complete    bool         `json:"complete"`
}

type UpsertProfileRequest struct {
	FirstName    string   `json:"firstName"`
	BirthDate    string   `json:"birthDate"`
	Gender       Gender   `json:"gender"`
	Bio          string   `json:"bio"`
	Interests    []string `json:"interests"`
	Discoverable *bool    `json:"discoverable"`
	ShowDistance *bool    `json:"showDistance"`
}

type PreferencesRequest struct {
	InterestedIn  []Gender `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	City      string   `json:"city"`
}

// ProfileInput is the validated, normalized profile data handed to the repository.
type ProfileInput struct {
	FirstName    string
	BirthDate    time.Time
	Gender       Gender
	Bio          string
	InterestSlug []string
	Discoverable bool
	ShowDistance bool
}

// ProfileHeader is the lightweight card used in lists (matches, conversations, blocks).
type ProfileSummary struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	ThumbURL  *string `json:"thumbUrl"`
}
