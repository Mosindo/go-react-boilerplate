package photos

import "errors"

const (
	MaxPhotosPerUser = 6
	MaxUploadBytes   = 8 << 20
	fullMaxSide      = 1080
	thumbMaxSide     = 320
	maxSourceSide    = 10000
	maxSourcePixels  = 50_000_000
	jpegQuality      = 82
)

var (
	ErrInvalidImage  = errors.New("file is not a valid JPEG or PNG image")
	ErrPhotoLimit    = errors.New("maximum number of photos reached")
	ErrNotFound      = errors.New("photo not found")
	ErrInvalidOrder  = errors.New("photoIds must list exactly your photos")
	ErrImageTooLarge = errors.New("image is too large")
)

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,max=6"`
}
