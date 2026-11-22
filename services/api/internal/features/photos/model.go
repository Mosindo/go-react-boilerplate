package photos

import (
	"errors"

	"example.com/api/internal/features/profiles"
)

var (
	ErrPhotoNotFound  = errors.New("photo not found")
	ErrTooManyPhotos  = errors.New("photo limit reached")
	ErrInvalidOrder   = errors.New("order must list each of your photos exactly once")
	ErrMissingPayload = errors.New("photo file is required")
)

type PhotosResponse struct {
	Photos []profiles.Photo `json:"photos"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6,dive,uuid"`
}

type storedPhoto struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
	Width      int
	Height     int
}
