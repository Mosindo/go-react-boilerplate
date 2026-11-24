package photos

import "time"

const MaxPhotosPerUser = 6

type Photo struct {
	ID         string
	UserID     string
	Position   int
	StorageKey string
	Width      int
	Height     int
	SizeBytes  int
	CreatedAt  time.Time
}

// PhotoResponse is the public representation of a photo. URL is relative to
// the API base URL, signed and expiring.
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
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6,dive,uuid"`
}
