package photos

import "example.com/api/internal/features/profiles"

const MaxPhotos = 6

type StoredPhoto struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
	Width      int
	Height     int
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6"`
}

type PhotosResponse struct {
	Photos []profiles.Photo `json:"photos"`
}
