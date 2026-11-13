package photos

import "time"

const MaxPhotosPerUser = 6

type Photo struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Position  int       `json:"position"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"createdAt"`
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
	CreatedAt  time.Time
}
