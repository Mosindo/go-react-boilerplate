package users

import "time"

const (
	GenderMan       = "man"
	GenderWoman     = "woman"
	GenderNonBinary = "non_binary"

	MinAge = 18
	MaxAge = 99
)

var validGenders = map[string]bool{GenderMan: true, GenderWoman: true, GenderNonBinary: true}

type Interest struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID        string `json:"id"`
	Position  int    `json:"position"`
	IsPrimary bool   `json:"isPrimary"`
	URL       string `json:"url"`
}

type PublicProfile struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
	DistanceKm *int       `json:"distanceKm"`
	LikedYou   bool       `json:"likedYou"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm *int     `json:"maxDistanceKm"`
}

type OwnProfile struct {
	UserID       string      `json:"userId"`
	FirstName    string      `json:"firstName"`
	Age          int         `json:"age"`
	BirthDate    string      `json:"birthDate"`
	Gender       string      `json:"gender"`
	Bio          string      `json:"bio"`
	City         string      `json:"city"`
	Interests    []Interest  `json:"interests"`
	Photos       []Photo     `json:"photos"`
	Discoverable bool        `json:"discoverable"`
	ShowDistance bool        `json:"showDistance"`
	HasLocation  bool        `json:"hasLocation"`
	Preferences  Preferences `json:"preferences"`
}

type ProfileInput struct {
	FirstName   string `json:"firstName"`
	BirthDate   string `json:"birthDate"`
	Gender      string `json:"gender"`
	Bio         string `json:"bio"`
	City        string `json:"city"`
	InterestIDs []int  `json:"interestIds"`
}

type Privacy struct {
	Discoverable *bool `json:"discoverable" binding:"required"`
	ShowDistance *bool `json:"showDistance" binding:"required"`
}

type LocationInput struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
}

// storedProfile is the raw database projection of a profile.
type storedProfile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Age          int
	Gender       string
	Bio          string
	City         string
	Discoverable bool
	ShowDistance bool
	HasLocation  bool
}

// AgeOn returns the full years elapsed between birth and now.
func AgeOn(birth, now time.Time) int {
	age := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		age--
	}
	return age
}
