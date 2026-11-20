package photos

import "time"

const (
	MaxPhotosPerUser = 6
	// MaxUploadBytes bounds the request body; larger uploads are rejected
	// before any decoding happens.
	MaxUploadBytes  = 10 << 20
	maxSourcePixels = 40_000_000
	maxSourceSide   = 10_000
	minSourceSide   = 200
	maxOutputSide   = 1440
	jpegQuality     = 82
)

type Photo struct {
	ID        string
	UserID    string
	Position  int
	URL       string
	Width     int
	Height    int
	CreatedAt time.Time
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
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6,dive,uuid"`
}

func toResponse(p Photo) PhotoResponse {
	return PhotoResponse{ID: p.ID, URL: p.URL, Position: p.Position, Width: p.Width, Height: p.Height}
}

func toResponses(list []Photo) PhotosResponse {
	out := make([]PhotoResponse, 0, len(list))
	for _, p := range list {
		out = append(out, toResponse(p))
	}
	return PhotosResponse{Photos: out}
}
