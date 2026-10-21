package discovery

import "example.com/api/internal/features/profiles"

type Page struct {
	Items []profiles.PublicProfile `json:"items"`
}

type candidate struct {
	UserID     string
	DistanceKm *float64
}
