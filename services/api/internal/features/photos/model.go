package photos

import "example.com/api/internal/features/profiles"

const MaxPhotos = 6

type StoredPhoto struct {
	ID         string
	UserID     string
	Position   int
	StorageKey string
	ThumbKey   string
	Width      int
	Height     int
	SizeBytes  int
}

type OrderRequest struct {
	IDs []string `json:"ids"`
}

type PhotoRef = profiles.Photo
