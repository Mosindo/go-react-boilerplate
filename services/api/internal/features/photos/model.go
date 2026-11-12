package photos

import "errors"

const MaxPhotos = 6

var (
	ErrProfileRequired = errors.New("create your profile before adding photos")
	ErrTooManyPhotos   = errors.New("you can upload up to 6 photos")
	ErrPhotoNotFound   = errors.New("photo not found")
	ErrInvalidOrder    = errors.New("photo order must list each of your photos exactly once")
)

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Position int    `json:"position"`
}

type PhotosResponse struct {
	Photos []Photo `json:"photos"`
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6"`
}

type storedPhoto struct {
	ID         string
	StorageKey string
	SizeBytes  int
	Width      int
	Height     int
	Position   int
}

type newPhoto struct {
	StorageKey string
	SizeBytes  int
	Width      int
	Height     int
}
