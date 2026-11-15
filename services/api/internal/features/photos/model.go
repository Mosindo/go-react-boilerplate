package photos

import "time"

const (
	MaxPhotos         = 6
	MaxUploadBytes    = 10 << 20 // 10 MiB raw upload
	MinDimension      = 200
	MaxDimension      = 10000
	MaxSourcePixels   = 40_000_000 // guards against decompression bombs
	OutputMaxSide     = 1600
	OutputJPEGQuality = 85
)

type Photo struct {
	ID         string
	UserID     string
	StorageKey string
	Width      int
	Height     int
	SizeBytes  int
	Position   int
	CreatedAt  time.Time
}

type PhotoResponse struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Position int    `json:"position"`
}

type PhotosResponse struct {
	Photos []PhotoResponse `json:"photos"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6,dive,uuid"`
}

// ProcessedImage is a sanitised, re-encoded JPEG ready for storage.
type ProcessedImage struct {
	Data   []byte
	Width  int
	Height int
}
