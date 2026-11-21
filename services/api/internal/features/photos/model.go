package photos

import "example.com/api/internal/features/profiles"

type StoredPhoto struct {
	ID              string
	UserID          string
	StorageKey      string
	ThumbStorageKey string
	Position        int
	Width           int
	Height          int
	// SizeBytes is only used when writing (size of the full variant).
	SizeBytes int
}

type PhotosResponse struct {
	Photos []profiles.PhotoResponse `json:"photos"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6"`
}
