package profiles

import "time"

type Gender string

const (
	GenderMan       Gender = "man"
	GenderWoman     Gender = "woman"
	GenderNonBinary Gender = "nonbinary"
)

var allGenders = []Gender{GenderMan, GenderWoman, GenderNonBinary}

func (g Gender) Valid() bool {
	for _, v := range allGenders {
		if g == v {
			return true
		}
	}
	return false
}

const (
	MinAge = 18
	MaxAge = 99
)

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

// PhotoURL is the authenticated path serving a photo's bytes.
func PhotoURL(photoID string) string { return "/photos/" + photoID + "/file" }

// ProfileSummary is the compact identity shown in lists (matches, chats, blocks).
type ProfileSummary struct {
	ID           string  `json:"id"`
	FirstName    string  `json:"firstName"`
	Age          int     `json:"age"`
	MainPhotoURL *string `json:"mainPhotoUrl"`
}

// PublicProfile is what other users may see. It never contains e-mail, birth
// date or coordinates.
type PublicProfile struct {
	ID         string     `json:"id"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     Gender     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
}

type MyProfile struct {
	FirstName    string     `json:"firstName"`
	BirthDate    string     `json:"birthDate"`
	Age          int        `json:"age"`
	Gender       Gender     `json:"gender"`
	Bio          string     `json:"bio"`
	City         string     `json:"city"`
	Latitude     float64    `json:"latitude"`
	Longitude    float64    `json:"longitude"`
	Interests    []Interest `json:"interests"`
	Photos       []Photo    `json:"photos"`
	Discoverable bool       `json:"discoverable"`
	ShowDistance bool       `json:"showDistance"`
}

type Preferences struct {
	InterestedIn  []Gender `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type OnboardingState struct {
	Profile     bool `json:"profile"`
	Photos      bool `json:"photos"`
	Preferences bool `json:"preferences"`
	Complete    bool `json:"complete"`
}

type MeResponse struct {
	ID         string          `json:"id"`
	Email      string          `json:"email"`
	CreatedAt  time.Time       `json:"createdAt"`
	Onboarding OnboardingState `json:"onboarding"`
}

type UpdateProfileRequest struct {
	FirstName    string   `json:"firstName"`
	BirthDate    string   `json:"birthDate"`
	Gender       Gender   `json:"gender"`
	Bio          string   `json:"bio"`
	City         string   `json:"city"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	Interests    []string `json:"interests"`
	Discoverable *bool    `json:"discoverable"`
	ShowDistance *bool    `json:"showDistance"`
}

// ProfileInput is a validated, normalised profile ready to be stored.
type ProfileInput struct {
	FirstName     string
	BirthDate     time.Time
	Gender        Gender
	Bio           string
	City          string
	Latitude      float64
	Longitude     float64
	InterestSlugs []string
	Discoverable  *bool
	ShowDistance  *bool
}
