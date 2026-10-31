package profiles

import "time"

const (
	GenderWoman     = "woman"
	GenderMan       = "man"
	GenderNonbinary = "nonbinary"

	MaxInterests = 10
	MaxBioRunes  = 500
	MaxNameRunes = 40
	MaxCityRunes = 80
)

var validGenders = map[string]bool{GenderWoman: true, GenderMan: true, GenderNonbinary: true}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type PhotoRef struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Card is what other users may see. It never contains email, birth date,
// coordinates or any exact-location data.
type Card struct {
	ID         string     `json:"id"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm,omitempty"`
	Interests  []Interest `json:"interests"`
	Photos     []PhotoRef `json:"photos"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// OwnProfile is the private view returned to the owner.
type OwnProfile struct {
	UserID         string      `json:"userId"`
	FirstName      string      `json:"firstName"`
	Age            int         `json:"age"`
	Gender         string      `json:"gender"`
	Bio            string      `json:"bio"`
	City           string      `json:"city"`
	Location       *Location   `json:"location"`
	IsDiscoverable bool        `json:"isDiscoverable"`
	ShowDistance   bool        `json:"showDistance"`
	Interests      []Interest  `json:"interests"`
	Photos         []PhotoRef  `json:"photos"`
	Preferences    Preferences `json:"preferences"`
	Complete       bool        `json:"complete"`
	Exists         bool        `json:"exists"`
}

type UpdateProfileRequest struct {
	FirstName      string    `json:"firstName"`
	Gender         string    `json:"gender"`
	Bio            string    `json:"bio"`
	City           string    `json:"city"`
	Location       *Location `json:"location"`
	Interests      []string  `json:"interests"`
	IsDiscoverable *bool     `json:"isDiscoverable"`
	ShowDistance   *bool     `json:"showDistance"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

// ProfileInput is the validated, normalized form passed to the repository.
type ProfileInput struct {
	FirstName      string
	Gender         string
	Bio            string
	City           string
	Location       *Location
	InterestSlugs  []string
	IsDiscoverable bool
	ShowDistance   bool
}

type storedProfile struct {
	UserID         string
	FirstName      string
	BirthDate      time.Time
	Gender         string
	Bio            string
	City           string
	Location       *Location
	IsDiscoverable bool
	ShowDistance   bool
}

func photoURL(id string) string { return "/photos/" + id }
