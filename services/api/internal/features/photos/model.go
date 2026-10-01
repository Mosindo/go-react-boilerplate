package photos

import "time"

const (
	MaxPhotos         = 6
	MaxUploadBytes    = 5 << 20 // 5 MiB
	MaxPixels         = 40_000_000
	MinSidePx         = 200
	MaxSidePx         = 12_000
	OutputMaxSidePx   = 1080
	OutputJPEGQuality = 85
)

type Photo struct {
	ID        string    `json:"id"`
	Position  int       `json:"position"`
	URL       string    `json:"url"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"createdAt"`
}

type PhotosResponse struct {
	Photos []Photo `json:"photos"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds"`
}

// stored is the persistence shape of a photo.
type stored struct {
	ID         string
	UserID     string
	Position   int
	StorageKey string
	SizeBytes  int
	Width      int
	Height     int
	CreatedAt  time.Time
}

func (s stored) toPhoto() Photo {
	return Photo{ID: s.ID, Position: s.Position, URL: "/photos/" + s.ID + "/file", Width: s.Width, Height: s.Height, CreatedAt: s.CreatedAt}
}
