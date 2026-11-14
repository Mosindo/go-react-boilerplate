package photos

import "time"

const MaxPhotos = 6

type Photo struct {
	ID         string
	UserID     string
	StorageKey string
	SizeBytes  int
	Width      int
	Height     int
	Position   int
	CreatedAt  time.Time
}

type PhotoResponse struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type PhotosResponse struct {
	Photos []PhotoResponse `json:"photos"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,max=6"`
}
