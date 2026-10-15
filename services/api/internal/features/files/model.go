package files

import "time"

const (
	// MaxPhotos is the number of photos a profile can hold.
	MaxPhotos = 6
	// MaxUploadBytes bounds the raw upload; clients compress well below this.
	MaxUploadBytes = 10 << 20

	// Decompression-bomb guards, checked on the image header before decoding.
	maxDimension = 12000
	maxPixels    = 40_000_000
	minDimension = 200

	fullSize  = 1080
	thumbSize = 360

	jpegQuality = 82
)

type Photo struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
	Width      int
	Height     int
	SizeBytes  int
	CreatedAt  time.Time
}

type ReorderRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=6,dive,uuid"`
}
