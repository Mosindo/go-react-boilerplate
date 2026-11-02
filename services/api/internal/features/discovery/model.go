package discovery

import "example.com/api/internal/features/profiles"

// Card is the public view of another user's profile. It never contains
// coordinates, birth date or e-mail; distance is bucketed.
type Card struct {
	UserID          string              `json:"userId"`
	FirstName       string              `json:"firstName"`
	Age             int                 `json:"age"`
	Gender          string              `json:"gender"`
	Bio             string              `json:"bio"`
	City            string              `json:"city"`
	DistanceKm      *int                `json:"distanceKm"`
	Interests       []profiles.Interest `json:"interests"`
	SharedInterests int                 `json:"sharedInterests"`
	Photos          []profiles.PhotoRef `json:"photos"`
}

type SwipeResult struct {
	Action  string `json:"action"`
	Matched bool   `json:"matched"`
	MatchID string `json:"matchId,omitempty"`
}
