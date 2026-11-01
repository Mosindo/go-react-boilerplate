package photos

import (
	"errors"
	"time"
)

var (
	ErrNotFound     = errors.New("photo not found")
	ErrLimitReached = errors.New("photo limit reached")
	ErrInvalidImage = errors.New("invalid image")
	ErrTooLarge     = errors.New("image too large")
	ErrInvalidOrder = errors.New("invalid order")
)

const (
	MaxPhotos      = 6
	MaxUploadBytes = 8 << 20 // 8 Mo avant recompression
	MaxPixels      = 40_000_000
	MaxSide        = 1080
	JPEGQuality    = 82
)

type Photo struct {
	ID        string    `json:"id"`
	Position  int       `json:"position"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"createdAt"`
}

// File decrit un fichier stocke.
type File struct {
	StorageKey  string
	ContentType string
}

type OrderRequest struct {
	IDs []string `json:"ids" binding:"required,min=1,max=6"`
}
