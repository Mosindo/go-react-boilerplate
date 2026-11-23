package photos

import "time"

const (
	MaxPhotos       = 6
	MaxUploadBytes  = 10 << 20 // 10 MiB raw upload
	MaxSourcePixels = 40_000_000
	MaxDimension    = 1600 // long edge after processing
	JPEGQuality     = 85
)

type StoredPhoto struct {
	ID         string
	UserID     string
	Position   int
	StorageKey string
	SizeBytes  int
	Width      int
	Height     int
	CreatedAt  time.Time
}

type PhotoResponse struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
}

type PhotosResponse struct {
	Photos []PhotoResponse `json:"photos"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6"`
}

// ProcessedImage is a validated, re-encoded image ready for storage.
type ProcessedImage struct {
	Data   []byte
	Width  int
	Height int
}
