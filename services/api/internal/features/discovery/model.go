package discovery

import "time"

type Interest struct {
	Slug   string `json:"slug"`
	Label  string `json:"label"`
	Shared bool   `json:"shared"`
}

type Photo struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Card is everything another member may see about a profile. It never
// contains e-mail, birthdate, coordinates or moderation state.
type Card struct {
	UserID           string     `json:"userId"`
	FirstName        string     `json:"firstName"`
	Age              int        `json:"age"`
	Gender           string     `json:"gender"`
	Bio              string     `json:"bio"`
	JobTitle         string     `json:"jobTitle"`
	City             string     `json:"city"`
	HeightCm         *int       `json:"heightCm"`
	RelationshipGoal string     `json:"relationshipGoal"`
	DistanceKm       *int       `json:"distanceKm"`
	SharedInterests  int        `json:"sharedInterests"`
	Interests        []Interest `json:"interests"`
	Photos           []Photo    `json:"photos"`
	IsMatch          bool       `json:"isMatch"`
	IsDemo           bool       `json:"isDemo"`
}

type FeedResponse struct {
	Profiles []Card `json:"profiles"`
}

type storedCard struct {
	UserID           string
	FirstName        string
	Birthdate        time.Time
	Gender           string
	Bio              string
	JobTitle         string
	City             string
	HeightCm         *int16
	RelationshipGoal string
	DistanceKm       *float64
	ShowDistance     bool
	SharedInterests  int
	IsMatch          bool
	IsDemo           bool
}

type storedPhoto struct {
	UserID string
	ID     string
	Width  int
	Height int
}

type storedInterest struct {
	UserID string
	Slug   string
	Label  string
	Shared bool
}
