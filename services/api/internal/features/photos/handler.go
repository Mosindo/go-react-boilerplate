package photos

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig):
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "payload_too_large", "file is too large (8 MB max)")
	case errors.Is(err, ErrInvalidImage), errors.Is(err, ErrImageTooLarge):
		httpx.Fail(c, http.StatusBadRequest, "invalid_image", err.Error())
	case errors.Is(err, ErrPhotoLimit):
		httpx.Fail(c, http.StatusConflict, "photo_limit", err.Error())
	case errors.Is(err, ErrInvalidOrder):
		httpx.BadRequest(c, err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.NotFound(c)
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func readUpload(c *gin.Context) ([]byte, error) {
	header, err := c.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return nil, err
		}
		return nil, ErrInvalidImage
	}
	if header.Size > MaxUploadBytes {
		return nil, &http.MaxBytesError{Limit: MaxUploadBytes}
	}
	f, err := header.Open()
	if err != nil {
		return nil, ErrInvalidImage
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxUploadBytes+1))
	if err != nil {
		return nil, ErrInvalidImage
	}
	if len(data) > MaxUploadBytes {
		return nil, &http.MaxBytesError{Limit: MaxUploadBytes}
	}
	return data, nil
}

func (h *Handler) Add(c *gin.Context) {
	data, err := readUpload(c)
	if err != nil {
		h.fail(c, "photos.add.read", err)
		return
	}
	photo, err := h.service.Add(c.Request.Context(), c.GetString("userID"), data)
	if err != nil {
		h.fail(c, "photos.add", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Replace(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	data, err := readUpload(c)
	if err != nil {
		h.fail(c, "photos.replace.read", err)
		return
	}
	photo, err := h.service.Replace(c.Request.Context(), c.GetString("userID"), id, data)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, photo)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	for i, id := range req.PhotoIDs {
		if !httpx.IsUUID(id) {
			httpx.BadRequest(c, "photoIds must be UUIDs")
			return
		}
		req.PhotoIDs[i] = lower(id)
	}
	items, err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.PhotoIDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Image(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	size := c.DefaultQuery("size", "full")
	if size != "full" && size != "thumb" {
		httpx.BadRequest(c, "size must be full or thumb")
		return
	}
	data, err := h.service.Image(c.Request.Context(), c.GetString("userID"), id, size == "thumb")
	if err != nil {
		h.fail(c, "photos.image", err)
		return
	}
	// Photo ids are immutable (replacing a photo creates a new id), so the digest is stable per id+size.
	sum := sha256.Sum256(append([]byte(id+size), data[:min(len(data), 64)]...))
	etag := `"` + hex.EncodeToString(sum[:8]) + `"`
	c.Header("ETag", etag)
	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, "image/jpeg", data)
}

func lower(s string) string {
	b := []byte(s)
	for i, ch := range b {
		if ch >= 'A' && ch <= 'F' {
			b[i] = ch + 32
		}
	}
	return string(b)
}
