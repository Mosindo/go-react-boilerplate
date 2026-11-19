package photos

import (
	"errors"
	"io"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/media"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: items})
}

func (h *Handler) Upload(c *gin.Context) {
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	photo, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), raw)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Replace(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	items, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), id, raw)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: items})
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	items, err := h.service.Delete(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: items})
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "photoIds must list 1 to 6 photo ids")
		return
	}
	items, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: items})
}

// Media serves a stored photo behind a signed, expiring URL.
func (h *Handler) Media(c *gin.Context) {
	path, err := h.service.ResolveMedia(c.Request.Context(), c.Param("key"), c.Query("exp"), c.Query("sig"))
	if errors.Is(err, ErrMediaNotFound) || errors.Is(err, media.ErrInvalidKey) {
		httpx.Fail(c, http.StatusNotFound, "not_found", "media not found")
		return
	}
	if err != nil {
		httpx.Internal(c, "photos.media", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600, immutable")
	c.Header("Content-Type", "image/jpeg")
	// Images are embedded by the web client from another origin.
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.File(path)
}

func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, media.MaxUploadBytes+64<<10)
	file, _, err := c.Request.FormFile("photo")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "photos must be 8 MB or smaller")
			return nil, false
		}
		httpx.BadRequest(c, "a multipart field named \"photo\" is required")
		return nil, false
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, media.MaxUploadBytes+1))
	if err != nil {
		httpx.BadRequest(c, "could not read upload")
		return nil, false
	}
	if len(raw) > media.MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "photos must be 8 MB or smaller")
		return nil, false
	}
	return raw, true
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "photo_not_found", "photo not found")
	case errors.Is(err, ErrLimitReached):
		httpx.Fail(c, http.StatusConflict, "photo_limit", err.Error())
	case errors.Is(err, ErrLastPhoto):
		httpx.Fail(c, http.StatusConflict, "last_photo", err.Error())
	case errors.Is(err, ErrInvalidOrder):
		httpx.BadRequest(c, err.Error())
	case errors.Is(err, media.ErrUnsupportedType):
		httpx.Fail(c, http.StatusUnsupportedMediaType, "unsupported_type", err.Error())
	case errors.Is(err, media.ErrTooLarge):
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", err.Error())
	case errors.Is(err, media.ErrTooSmall), errors.Is(err, media.ErrCorrupt):
		httpx.BadRequest(c, err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}
