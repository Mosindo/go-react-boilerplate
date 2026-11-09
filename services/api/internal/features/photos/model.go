package photos

import "errors"

const (
	MaxPhotos     = 6
	MaxUploadSize = 8 << 20 // 8 MiB raw upload
	MaxPixels     = 40_000_000
	MaxDimension  = 1280 // stored images are downscaled to this longest side
	JPEGQuality   = 82
)

var (
	ErrNotFound      = errors.New("photo not found")
	ErrLimitReached  = errors.New("photo limit reached")
	ErrInvalidImage  = errors.New("invalid image")
	ErrOrderMismatch = errors.New("order must list every photo exactly once")
)

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type StoredPhoto struct {
	ID         string
	StorageKey string
	Position   int
	Width      int
	Height     int
	SizeBytes  int
}

type OrderRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=6"`
}
