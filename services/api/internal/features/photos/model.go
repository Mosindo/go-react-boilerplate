package photos

import "errors"

const MaxPhotos = 6

var (
	ErrNotFound    = errors.New("photo not found")
	ErrLimit       = errors.New("photo limit reached")
	ErrBadOrder    = errors.New("order must list every photo exactly once")
	ErrTooLarge    = errors.New("file too large")
	ErrEmptyUpload = errors.New("empty upload")
)

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6"`
}

type stored struct {
	ID         string
	UserID     string
	Position   int
	StorageKey string
	Width      int
	Height     int
}
