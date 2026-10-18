package photos

import "time"

const (
	MaxPhotosPerUser = 6
	maxUploadBytes   = 8 << 20
)

// Photo is the API representation. URLs are relative to the API origin and
// carry a signed, expiring capability so that plain <img> tags can load them.
type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type PhotosResponse struct {
	Photos []Photo `json:"photos"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6,dive,uuid"`
}

type storedPhoto struct {
	ID         string
	UserID     string
	Position   int
	StorageKey string
	ThumbKey   string
	Width      int
	Height     int
	SizeBytes  int
	CreatedAt  time.Time
}
