package photos

import "time"

const (
	MaxPhotosPerUser = 6
	MaxUploadBytes   = 8 << 20 // 8 MiB
	MaxEdgePixels    = 1440
	minEdgePixels    = 200
	maxSourcePixels  = 24_000_000
	maxSourceEdge    = 8_000
	jpegQuality      = 82
	storedMime       = "image/jpeg"
)

type Photo struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Position  int       `json:"position"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"createdAt"`
}

type PhotosResponse struct {
	Photos []Photo `json:"photos"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6,dive,max=36"`
}
