package photos

import "example.com/api/internal/features/profiles"

const MaxPhotos = profiles.MaxPhotos

type ListResponse struct {
	Photos []profiles.Photo `json:"photos"`
}

type ReorderRequest struct {
	IDs []string `json:"ids"`
}
