package photos

import "errors"

const MaxPhotos = 6

var (
	ErrNotFound      = errors.New("photo not found")
	ErrLimitReached  = errors.New("photo limit reached")
	ErrInvalidOrder  = errors.New("order must list each of your photos exactly once")
	ErrFileRequired  = errors.New("file required")
	ErrFileTooLarge  = errors.New("file too large")
	ErrStorageFailed = errors.New("storage failure")
)

type Photo struct {
	ID       string `json:"id"`
	Position int    `json:"position"`
	URL      string `json:"url"`
	ThumbURL string `json:"thumbUrl"`
}

type storedPhoto struct {
	ID, UserID, StorageKey, ThumbKey string
	Position                         int
}

type OrderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required"`
}
