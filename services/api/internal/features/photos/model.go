package photos

import "time"

const (
	MaxPhotos       = 6
	MaxUploadBytes  = 8 << 20 // 8 MiB
	MaxPixels       = 40_000_000
	MaxSide         = 1280
	JPEGQuality     = 82
	minSide         = 200
	storedMime      = "image/jpeg"
	storedExtension = ".jpg"
)

type Photo struct {
	ID        string    `json:"id"`
	Position  int       `json:"position"`
	URL       string    `json:"url"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"createdAt"`
}

type ListResponse struct {
	Photos []Photo `json:"photos"`
}

type OrderRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=6"`
}

// Stored is the database row, including the private storage key.
type Stored struct {
	Photo
	StorageKey string
}
