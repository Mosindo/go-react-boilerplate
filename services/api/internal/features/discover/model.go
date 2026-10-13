package discover

import "example.com/api/internal/features/profiles"

const (
	DefaultLimit = 10
	MaxLimit     = 20
)

type Response struct {
	Profiles []profiles.PublicProfile `json:"profiles"`
}
