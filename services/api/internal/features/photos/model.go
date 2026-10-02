package photos

import "time"

const (
	MaxPhotosPerUser = 6
	MaxUploadBytes   = 10 << 20 // raw upload limit
	MaxOutputSide    = 1280
	MinSide          = 200
	maxPixels        = 40_000_000 // decompression-bomb guard
)

// View is what clients see: an id, a position and a signed, expiring URL path.
type View struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

type Stored struct {
	ID         string
	UserID     string
	Position   int
	StorageKey string
	CreatedAt  time.Time
}

type ReorderRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=6"`
}
