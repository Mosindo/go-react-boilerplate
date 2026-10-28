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

type Interest struct {
	ID       int16  `json:"id"`
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Category string `json:"category"`
}

type Photo struct {
	ID       string `json:"id"`
	Position int16  `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	URL      string `json:"url"`
}

type ProfileInput struct {
	FirstName    string `json:"firstName" binding:"required,max=200"`
	BirthDate    string `json:"birthDate" binding:"required,max=10"`
	Gender       Gender `json:"gender" binding:"required,max=20"`
	Bio          string `json:"bio" binding:"max=4000"`
	City         string `json:"city" binding:"max=400"`
	InterestIDs  []int  `json:"interestIds" binding:"max=50"`
	ShowDistance *bool  `json:"showDistance"` // nil = keep current (true on creation)
	Discoverable *bool  `json:"discoverable"`
}

type LocationInput struct {
	Latitude  float64 `json:"latitude" binding:"required,min=-90,max=90"`
	Longitude float64 `json:"longitude" binding:"required,min=-180,max=180"`
	City      string  `json:"city" binding:"max=400"`
}

type Preferences struct {
	InterestedIn  []Gender `json:"interestedIn" binding:"required,min=1,max=3"`
	MinAge        int      `json:"minAge" binding:"required,min=18,max=99"`
	MaxAge        int      `json:"maxAge" binding:"required,min=18,max=99"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required,min=1,max=500"`
}

// OwnProfile is what a user sees about themselves (includes private settings and birth date).
type OwnProfile struct {
	UserID       string     `json:"userId"`
	FirstName    string     `json:"firstName"`
	BirthDate    string     `json:"birthDate"`
	Age          int        `json:"age"`
	Gender       Gender     `json:"gender"`
	Bio          string     `json:"bio"`
	City         string     `json:"city"`
	HasLocation  bool       `json:"hasLocation"`
	ShowDistance bool       `json:"showDistance"`
	Discoverable bool       `json:"discoverable"`
	Interests    []Interest `json:"interests"`
	Photos       []Photo    `json:"photos"`
}

// PublicProfile is what other users see. It never carries coordinates or the birth date.
type PublicProfile struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     Gender     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
}

type storedProfile struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Age          int
	Gender       Gender
	Bio          string
	City         string
	HasLocation  bool
	ShowDistance bool
	Discoverable bool
}
