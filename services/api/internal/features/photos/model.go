package photos

import "time"

const (
	MaxPhotosPerUser = 6
	MaxUploadBytes   = 10 << 20 // 10 MiB raw upload
	MaxSourcePixels  = 40_000_000
	MaxSourceSide    = 10_000
	MinSourceSide    = 200
	OutputMaxSide    = 1600
	OutputQuality    = 82
)

type Photo struct {
	ID         string
	UserID     string
	Position   int
	StorageKey string
	Width      int
	Height     int
	SizeBytes  int
	CreatedAt  time.Time
}

type PhotoResponse struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type PhotosResponse struct {
	Photos []PhotoResponse `json:"photos"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6"`
}
