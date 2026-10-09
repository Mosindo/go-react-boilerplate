package photos

import "example.com/api/internal/features/profiles"

type Photo = profiles.Photo

type StoredPhoto struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
}

type PhotosResponse struct {
	Photos []Photo `json:"photos"`
}

type OrderRequest struct {
	IDs []string `json:"ids" binding:"required,max=6"`
}

const (
	MaxPhotosPerUser = 6
	MaxUploadBytes   = 8 << 20
	maxSide          = 1440
	minSide          = 200
	maxSourcePixels  = 40_000_000
	jpegQuality      = 85
)
