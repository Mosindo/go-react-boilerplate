package photos

import "errors"

const MaxPhotos = 6

var (
	ErrNotFound      = errors.New("photo not found")
	ErrLimitReached  = errors.New("you can upload at most 6 photos")
	ErrInvalidOrder  = errors.New("order must list each of your photos exactly once")
	ErrLastPhoto     = errors.New("keep at least one photo while your profile is visible")
	ErrMediaNotFound = errors.New("media not found")
)

type Photo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Position int    `json:"position"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
}

type StoredPhoto struct {
	ID         string
	StorageKey string
	Position   int
	Width      int
	Height     int
}

type ReorderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,min=1,max=6,dive,uuid"`
}

type PhotosResponse struct {
	Photos []Photo `json:"photos"`
}
