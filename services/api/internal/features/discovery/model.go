package discovery

import (
	"errors"
	"time"
)

var (
	ErrBadRequest        = errors.New("invalid request")
	ErrIncompleteProfile = errors.New("complete your profile first")
)

const (
	defaultLimit = 10
	maxLimit     = 20
	overfetch    = 3
	maxFetch     = 60
	earthRadius  = 6371.0 // km
)

// Photo and PublicProfile mirror docs/API.md. Email and coordinates are never part of them.
type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type PublicProfile struct {
	UserID     string   `json:"userId"`
	FirstName  string   `json:"firstName"`
	Age        *int     `json:"age"`
	Gender     string   `json:"gender"`
	Bio        string   `json:"bio"`
	City       string   `json:"city"`
	DistanceKm *int     `json:"distanceKm"`
	Interests  []string `json:"interests"`
	Photos     []Photo  `json:"photos"`
}

// Candidate is a discovery row before it is turned into a PublicProfile.
type Candidate struct {
	UserID          string
	FirstName       string
	Gender          string
	Bio             string
	City            string
	Age             int
	ShowAge         bool
	ShowDistance    bool
	DistanceKm      float64 // exact, used for ranking only
	SharedInterests int
	LastActiveAt    time.Time
}

type viewer struct {
	ID           string
	Gender       string
	Lat, Lon     float64
	InterestedIn []string
	AgeMin       int
	AgeMax       int
	MaxDistance  int
	Age          int
}
