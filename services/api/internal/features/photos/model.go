package photos

import (
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
)

const MaxPhotos = 6

var (
	ErrInvalidImage = platformerrors.New(http.StatusBadRequest, "invalid_image", "the file must be a JPEG or PNG image between 200px and 8000px, up to 8 MB")
	ErrPhotoLimit   = platformerrors.New(http.StatusConflict, "photo_limit", "you can upload at most 6 photos")
	ErrNotFound     = platformerrors.New(http.StatusNotFound, "not_found", "photo not found")
	ErrBadOrder     = platformerrors.New(http.StatusBadRequest, "invalid_request", "photoIds must list every photo exactly once")
)

type Photo struct {
	ID        string `json:"id"`
	Position  int    `json:"position"`
	IsPrimary bool   `json:"isPrimary"`
	URL       string `json:"url"`
}

type OrderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,max=6"`
}
