package photos

import (
	"errors"
	"io"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

// maxUploadBytes bounds the raw upload; the server re-encodes everything to ~1080px JPEG.
const maxUploadBytes = 8 << 20

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "photo not found", err)
	case errors.Is(err, ErrLimit):
		httpx.Fail(c, op, http.StatusConflict, "photo limit reached (6)", err)
	case errors.Is(err, ErrBadOrder):
		httpx.Fail(c, op, http.StatusUnprocessableEntity, err.Error(), err)
	case errors.Is(err, ErrUnsupportedImage), errors.Is(err, ErrEmptyUpload):
		httpx.Fail(c, op, http.StatusUnsupportedMediaType, "unsupported image (use JPEG, PNG or WebP)", err)
	case errors.Is(err, ErrImageTooLarge), errors.Is(err, ErrTooLarge):
		httpx.Fail(c, op, http.StatusRequestEntityTooLarge, "image too large", err)
	default:
		httpx.Internal(c, op, err)
	}
}

func readUpload(c *gin.Context) ([]byte, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+(64<<10))
	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return nil, ErrTooLarge
		}
		return nil, ErrEmptyUpload
	}
	defer file.Close()
	if header.Size > maxUploadBytes {
		return nil, ErrTooLarge
	}
	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxUploadBytes {
		return nil, ErrTooLarge
	}
	return data, nil
}

func (h *Handler) Upload(c *gin.Context) {
	data, err := readUpload(c)
	if err != nil {
		h.fail(c, "photos.upload.read", err)
		return
	}
	p, err := h.service.Upload(c.Request.Context(), c.GetString("userID"), data)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) Replace(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	data, err := readUpload(c)
	if err != nil {
		h.fail(c, "photos.replace.read", err)
		return
	}
	p, err := h.service.Replace(c.Request.Context(), c.GetString("userID"), id, data)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if !httpx.BindJSON(c, "photos.reorder", &req) {
		return
	}
	for _, id := range req.PhotoIDs {
		if !httpx.IsUUID(id) {
			httpx.Fail(c, "photos.reorder", http.StatusUnprocessableEntity, "invalid photo id", nil)
			return
		}
	}
	if err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.PhotoIDs); err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Serve streams a photo to an authenticated viewer allowed to see it. Ids are immutable
// (replacing a photo creates a new id), so responses can be cached privately for a long time.
func (h *Handler) Serve(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	f, err := h.service.Open(c.Request.Context(), c.GetString("userID"), id, c.Query("size") == "thumb")
	if err != nil {
		h.fail(c, "photos.serve", err)
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.Header("Content-Disposition", "inline")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
