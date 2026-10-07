package profiles

import (
	"time"

	"example.com/api/internal/platform/media"
)

var validGenders = map[string]struct{}{"man": {}, "woman": {}, "nonbinary": {}}

type Interest struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

// UserSummary is the minimal public identity used in lists (matches, chats, blocks).
type UserSummary struct {
	UserID    string       `json:"userId"`
	FirstName string       `json:"firstName"`
	Age       int          `json:"age"`
	Photo     *media.Photo `json:"photo"`
}

type SelfProfile struct {
	UserID      string        `json:"userId"`
	FirstName   string        `json:"firstName"`
	BirthDate   string        `json:"birthDate"`
	Age         int           `json:"age"`
	Gender      string        `json:"gender"`
	Bio         string        `json:"bio"`
	City        string        `json:"city"`
	HasLocation bool          `json:"hasLocation"`
	Interests   []Interest    `json:"interests"`
	Photos      []media.Photo `json:"photos"`
	Complete    bool          `json:"complete"`
}

type Preferences struct {
	InterestedIn  []string `json:"interestedIn"`
	MinAge        int      `json:"minAge"`
	MaxAge        int      `json:"maxAge"`
	MaxDistanceKm int      `json:"maxDistanceKm"`
	Discoverable  bool     `json:"discoverable"`
	ShowDistance  bool     `json:"showDistance"`
}

func DefaultPreferences() Preferences {
	return Preferences{
		InterestedIn:  []string{"man", "woman", "nonbinary"},
		MinAge:        18,
		MaxAge:        99,
		MaxDistanceKm: 50,
		Discoverable:  true,
		ShowDistance:  true,
	}
}

type UpsertProfileRequest struct {
	FirstName string   `json:"firstName" binding:"required"`
	BirthDate string   `json:"birthDate" binding:"required"`
	Gender    string   `json:"gender" binding:"required"`
	Bio       string   `json:"bio"`
	City      string   `json:"city"`
	Interests []string `json:"interests"`
}

type LocationRequest struct {
	Latitude  *float64 `json:"latitude" binding:"required"`
	Longitude *float64 `json:"longitude" binding:"required"`
}

// StoredProfile is the persisted row (no derived fields).
type StoredProfile struct {
	UserID      string
	FirstName   string
	BirthDate   time.Time
	Gender      string
	Bio         string
	City        string
	HasLocation bool
}
