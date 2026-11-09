package profiles

import "time"

type Interest struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn" binding:"required,min=1,max=3"`
	MinAge        int      `json:"minAge" binding:"required"`
	MaxAge        int      `json:"maxAge" binding:"required"`
	MaxDistanceKm int      `json:"maxDistanceKm" binding:"required"`
}

// OwnProfile is what a user sees about themselves.
type OwnProfile struct {
	UserID             string       `json:"userId"`
	FirstName          string       `json:"firstName"`
	BirthDate          string       `json:"birthDate"`
	Age                int          `json:"age"`
	Gender             string       `json:"gender"`
	Bio                string       `json:"bio"`
	City               string       `json:"city"`
	HasLocation        bool         `json:"hasLocation"`
	IsVisible          bool         `json:"isVisible"`
	ShowDistance       bool         `json:"showDistance"`
	Interests          []Interest   `json:"interests"`
	Photos             []Photo      `json:"photos"`
	Preferences        *Preferences `json:"preferences"`
	OnboardingComplete bool         `json:"onboardingComplete"`
	CreatedAt          time.Time    `json:"createdAt"`
}

// PublicProfile is what other users see. It never contains coordinates or birth date.
type PublicProfile struct {
	UserID     string     `json:"userId"`
	FirstName  string     `json:"firstName"`
	Age        int        `json:"age"`
	Gender     string     `json:"gender"`
	Bio        string     `json:"bio"`
	City       string     `json:"city"`
	DistanceKm *int       `json:"distanceKm,omitempty"`
	Interests  []Interest `json:"interests"`
	Photos     []Photo    `json:"photos"`
}

// ProfileInput is the validated payload for creating or updating a profile.
type ProfileInput struct {
	FirstName    string   `json:"firstName"`
	BirthDate    string   `json:"birthDate"`
	Gender       string   `json:"gender"`
	Bio          string   `json:"bio"`
	City         string   `json:"city"`
	InterestIDs  []int    `json:"interestIds"`
	IsVisible    *bool    `json:"isVisible"`
	ShowDistance *bool    `json:"showDistance"`
	Latitude     *float64 `json:"latitude"`
	Longitude    *float64 `json:"longitude"`
}

type LocationInput struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
}
