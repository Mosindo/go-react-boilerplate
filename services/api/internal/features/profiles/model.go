package profiles

import "time"

var (
	Genders = []string{"woman", "man", "non_binary"}
)

const (
	MinAge          = 18
	MaxAge          = 99
	MaxInterests    = 10
	MaxBioRunes     = 500
	MaxNameRunes    = 40
	MaxShortRunes   = 80
	MaxDistanceKm   = 20000
	MinPhotosNeeded = 1
)

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
}

// Card is the public view of a profile shown to other users. It never contains
// the birth date, coordinates or e-mail address.
type Card struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	Occupation string     `json:"occupation"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
}

// Own is the private profile returned to its owner.
type Own struct {
	FirstName    string `json:"firstName"`
	BirthDate    string `json:"birthDate"`
	Age          int    `json:"age"`
	Gender       string `json:"gender"`
	Bio          string `json:"bio"`
	Occupation   string `json:"occupation"`
	City         string `json:"city"`
	HasLocation  bool   `json:"hasLocation"`
	IsVisible    bool   `json:"isVisible"`
	ShowDistance bool   `json:"showDistance"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type MyProfile struct {
	Profile            *Own         `json:"profile"`
	Preferences        *Preferences `json:"preferences"`
	Interests          []Interest   `json:"interests"`
	Photos             []Photo      `json:"photos"`
	OnboardingComplete bool         `json:"onboardingComplete"`
	Missing            []string     `json:"missing"`
}

type UpdateProfileRequest struct {
	FirstName  string   `json:"firstName"`
	BirthDate  string   `json:"birthDate"`
	Gender     string   `json:"gender"`
	Bio        string   `json:"bio"`
	Occupation string   `json:"occupation"`
	City       string   `json:"city"`
	Interests  []string `json:"interests"`
}

type UpdatePreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn"`
	AgeMin        int      `json:"ageMin"`
	AgeMax        int      `json:"ageMax"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      *string `json:"city"`
}

type UpdatePrivacyRequest struct {
	IsVisible    *bool `json:"isVisible"`
	ShowDistance *bool `json:"showDistance"`
}

// StoredProfile is the persistence view of a profile row.
type StoredProfile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	Occupation   string
	City         string
	HasLocation  bool
	IsVisible    bool
	ShowDistance bool
	DistanceKm   *float64
	Age          int
}

type ProfileInput struct {
	FirstName  string
	BirthDate  time.Time
	Gender     string
	Bio        string
	Occupation string
	City       string
}
