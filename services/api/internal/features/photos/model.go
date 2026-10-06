package photos

import "errors"

const MaxPhotosPerUser = 6

var (
	ErrNotFound     = errors.New("photo not found")
	ErrLimitReached = errors.New("photo limit reached")
	ErrInvalidOrder = errors.New("order must list every photo exactly once")
)

type Photo struct {
	ID          string
	UserID      string
	Position    int
	StorageKey  string
	ContentType string
	SizeBytes   int
	Width       int
	Height      int
}

type PhotoResponse struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type PhotosResponse struct {
	Photos []PhotoResponse `json:"photos"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6"`
}

func toResponse(p Photo) PhotoResponse {
	return PhotoResponse{ID: p.ID, Position: p.Position, URL: "/photos/" + p.ID + "/content", Width: p.Width, Height: p.Height}
}
