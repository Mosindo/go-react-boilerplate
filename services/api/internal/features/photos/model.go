package photos

import (
	"fmt"
	"strconv"
	"time"
)

const (
	MaxPhotosPerUser = 6
	MaxUploadBytes   = 8 << 20
)

// Photo is the API representation. URLs are relative and carry a version so
// clients can cache aggressively and still see replacements.
type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

func NewPhoto(id string, position, width, height int, updatedAt time.Time) Photo {
	v := strconv.FormatInt(updatedAt.Unix(), 10)
	return Photo{
		ID:       id,
		Position: position,
		URL:      fmt.Sprintf("/photos/%s?v=%s", id, v),
		ThumbURL: fmt.Sprintf("/photos/%s?size=thumb&v=%s", id, v),
		Width:    width,
		Height:   height,
	}
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds"`
}

type Blob struct {
	Data      []byte
	UpdatedAt time.Time
}
