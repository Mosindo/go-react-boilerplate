package photos

import "time"

const MaxPhotosPerUser = 6

// Photo is the public representation of a profile photo. URL is a relative,
// signed and expiring path that clients prefix with the API base URL.
type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	URL      string `json:"url"`
}

type storedPhoto struct {
	ID         string
	UserID     string
	StorageKey string
	Position   int
	Width      int
	Height     int
	CreatedAt  time.Time
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6,dive,uuid"`
}

type PhotosResponse struct {
	Photos []Photo `json:"photos"`
}
