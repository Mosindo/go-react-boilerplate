package profiles

import (
	"math"
	"time"
)

const (
	GenderMan       = "man"
	GenderWoman     = "woman"
	GenderNonBinary = "non_binary"

	MinAge          = 18
	MaxAge          = 99
	MaxBioLength    = 500
	MaxInterests    = 10
	MaxNameLength   = 40
	MaxCityLength   = 80
	DefaultDistance = 50
)

var validGenders = map[string]struct{}{GenderMan: {}, GenderWoman: {}, GenderNonBinary: {}}

type Interest struct {
	ID    int16  `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type PhotoRef struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

// Profile is the owner's view of their own profile.
type Profile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	City         string
	HasLocation  bool
	IsVisible    bool
	ShowDistance bool
	Interests    []Interest
	Photos       []PhotoRef
}

// PublicProfile is what other users see. It never carries coordinates, birth
// date or email; distance is an approximation.
type PublicProfile struct {
	ID         string     `json:"id"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	Photos     []PhotoRef `json:"photos"`
}

type Preferences struct {
	InterestedIn  []string
	MinAge        int
	MaxAge        int
	MaxDistanceKm int
}

type UpdateProfileRequest struct {
	FirstName    string   `json:"firstName" binding:"required,max=100"`
	BirthDate    string   `json:"birthDate" binding:"required,max=10"`
	Gender       string   `json:"gender" binding:"required,max=20"`
	Bio          string   `json:"bio" binding:"max=2000"`
	City         string   `json:"city" binding:"max=200"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
	InterestIDs  []int16  `json:"interestIds" binding:"max=50"`
	IsVisible    *bool    `json:"isVisible"`
	ShowDistance *bool    `json:"showDistance"`
}

type PreferencesRequest struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,max=10"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

type ProfileResponse struct {
	UserID       string     `json:"userId"`
	FirstName    string     `json:"firstName"`
	BirthDate    string     `json:"birthDate"`
	Age          int        `json:"age"`
	Gender       string     `json:"gender"`
	Bio          string     `json:"bio"`
	City         string     `json:"city"`
	HasLocation  bool       `json:"hasLocation"`
	IsVisible    bool       `json:"isVisible"`
	ShowDistance bool       `json:"showDistance"`
	Interests    []Interest `json:"interests"`
	Photos       []PhotoRef `json:"photos"`
}

type PreferencesResponse struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
}

func PhotoURL(id string) string { return "/photos/" + id + "/file" }

// AgeAt returns the age in whole years on the given day.
func AgeAt(birth, now time.Time) int {
	years := now.Year() - birth.Year()
	if now.Month() < birth.Month() || (now.Month() == birth.Month() && now.Day() < birth.Day()) {
		years--
	}
	return years
}

// ApproxDistance converts a precise distance to a coarse value that does not
// reveal an exact position: 1 km steps up to 10 km, then 5 km steps.
func ApproxDistance(km float64) int {
	if km <= 1 {
		return 1
	}
	if km <= 10 {
		return int(km + 0.999999)
	}
	return ((int(math.Ceil(km)) + 4) / 5) * 5
}

func roundCoord(v float64) float64 {
	return float64(int64(v*100+sign(v)*0.5)) / 100
}

func sign(v float64) float64 {
	if v < 0 {
		return -1
	}
	return 1
}
