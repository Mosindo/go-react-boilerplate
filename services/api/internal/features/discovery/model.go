package discovery

import (
	"time"

	"example.com/api/internal/features/photos"
)

type CardInterest struct {
	Slug   string `json:"slug"`
	Label  string `json:"label"`
	Shared bool   `json:"shared"`
}

// Card is what other users see of a profile. It never contains exact
// coordinates, the birth date, or the e-mail address.
type Card struct {
	UserID     string         `json:"userId"`
	FirstName  string         `json:"firstName"`
	Age        int            `json:"age"`
	Gender     string         `json:"gender"`
	Bio        string         `json:"bio"`
	City       string         `json:"city"`
	DistanceKm *int           `json:"distanceKm"`
	Interests  []CardInterest `json:"interests"`
	Photos     []photos.Photo `json:"photos"`
}

// Candidate is a raw repository row before interests and photos are attached.
type Candidate struct {
	UserID       string
	FirstName    string
	BirthDate    time.Time
	Gender       string
	Bio          string
	City         string
	DistanceKm   *float64
	ShowDistance bool
}
