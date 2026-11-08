package photos

import "example.com/api/internal/features/profiles"

const (
	MaxUploadBytes = 8 << 20 // raw upload limit
	MaxPixels      = 40_000_000
	MinSide        = 200
	MaxOutputSide  = 1440
	JPEGQuality    = 82
)

type Photo = profiles.PhotoRef

type ReorderRequest struct {
	IDs []string `json:"ids"`
}

type stored struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
}
