package profiles

import "time"

const (
	GenderMan       = "man"
	GenderWoman     = "woman"
	GenderNonBinary = "non_binary"

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
	Position int    `json:"position"`
	URL      string `json:"url"`
}

type Profile struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
	DistanceKm *int       `json:"distanceKm"`
	BirthDate  string     `json:"birthDate,omitempty"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,min=1,max=3,dive,oneof=man woman non_binary"`
	MinAge        int      `json:"minAge" binding:"required,min=18,max=99"`
	MaxAge        int      `json:"maxAge" binding:"required,min=18,max=99"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required,min=1,max=500"`
	Discoverable  bool     `json:"discoverable"`
	ShowDistance  bool     `json:"showDistance"`
}

type ProfileInput struct {
	FirstName string   `json:"firstName" binding:"required,max=100"`
	BirthDate string   `json:"birthDate" binding:"required,max=10"`
	Gender    string   `json:"gender" binding:"required,max=20"`
	Bio       string   `json:"bio" binding:"max=2000"`
	City      string   `json:"city" binding:"max=200"`
	Interests []string `json:"interests" binding:"max=50,dive,max=50"`
}

type LocationInput struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
}

type MyProfileResponse struct {
	Profile     *Profile    `json:"profile"`
	Preferences Preferences `json:"preferences"`
	Complete    bool        `json:"complete"`
}

// storedProfile is the validated, normalized form written by the repository.
type storedProfile struct {
	FirstName string
	BirthDate time.Time
	Gender    string
	Bio       string
	City      string
}

func DefaultPreferences() Preferences {
	return Preferences{
		InterestedIn:  []string{GenderMan, GenderWoman, GenderNonBinary},
		MinAge:        MinAge,
		MaxAge:        MaxAge,
		MaxDistanceKm: 50,
		Discoverable:  true,
		ShowDistance:  true,
	}
}

// UserSummary is the compact identity shown in lists (matches, chats, notifications).
type UserSummary struct {
	UserID    string  `json:"userId"`
	FirstName string  `json:"firstName"`
	PhotoURL  *string `json:"photoUrl"`
}
