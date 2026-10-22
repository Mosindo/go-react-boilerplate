package photos

import (
	"errors"
	"io"
	"net/http"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

type orderRequest struct {
	PhotoIDs []string `json:"photoIds" binding:"required,max=6"`
}

func writeError(c *gin.Context, op string, err error) {
	var validation *platformerrors.ValidationError
	status := http.StatusInternalServerError
	body := gin.H{"error": "internal error"}
	switch {
	case errors.As(err, &validation):
		status, body = http.StatusBadRequest, gin.H{"error": validation.Message, "field": validation.Field}
	case errors.Is(err, ErrNotFound):
		status, body = http.StatusNotFound, gin.H{"error": "photo not found"}
	case errors.Is(err, ErrLimitReached):
		status, body = http.StatusConflict, gin.H{"error": "photo limit reached (6 maximum)"}
	case errors.Is(err, ErrLastPhoto):
		status, body = http.StatusConflict, gin.H{"error": "at least one photo is required"}
	case errors.Is(err, ErrConflict):
		status, body = http.StatusConflict, gin.H{"error": "photo list changed, retry"}
	case errors.Is(err, ErrBadOrder):
		status, body = http.StatusBadRequest, gin.H{"error": "order must list every photo exactly once"}
	case errors.Is(err, ErrUnsupportedImage):
		status, body = http.StatusUnsupportedMediaType, gin.H{"error": "unsupported image type (JPEG, PNG or WebP)"}
	case errors.Is(err, ErrImageTooLarge):
		status, body = http.StatusRequestEntityTooLarge, gin.H{"error": "image too large"}
	case errors.Is(err, ErrImageTooSmall):
		status, body = http.StatusBadRequest, gin.H{"error": "image too small (200px minimum)"}
	case errors.Is(err, ErrInvalidImage):
		status, body = http.StatusBadRequest, gin.H{"error": "invalid image"}
	}
	logger.LogHandlerError(c, op, status, err)
	c.JSON(status, body)
}

func (h *Handler) openUpload(c *gin.Context) (io.ReadCloser, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	header, err := c.FormFile("photo")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(c, "photos.upload", ErrImageTooLarge)
			return nil, false
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "multipart field \"photo\" is required"})
		return nil, false
	}
	f, err := header.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid upload"})
		return nil, false
	}
	return f, true
}

func (h *Handler) List(c *gin.Context) {
	photos, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		writeError(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

func (h *Handler) Upload(c *gin.Context) {
	f, ok := h.openUpload(c)
	if !ok {
		return
	}
	defer f.Close()
	photo, err := h.service.Upload(c.Request.Context(), c.GetString("userID"), f)
	if err != nil {
		writeError(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Replace(c *gin.Context) {
	id := c.Param("id")
	if !profiles.IsUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "photo not found"})
		return
	}
	f, ok := h.openUpload(c)
	if !ok {
		return
	}
	defer f.Close()
	photo, err := h.service.Replace(c.Request.Context(), c.GetString("userID"), id, f)
	if err != nil {
		writeError(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, photo)
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("id")
	if !profiles.IsUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "photo not found"})
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.GetString("userID"), id); err != nil {
		writeError(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req orderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	photos, err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.PhotoIDs)
	if err != nil {
		writeError(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

func (h *Handler) SetPrimary(c *gin.Context) {
	id := c.Param("id")
	if !profiles.IsUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "photo not found"})
		return
	}
	photos, err := h.service.SetPrimary(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		writeError(c, "photos.primary", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

// File streams a photo to an authorised viewer. Image URLs are immutable (a
// replacement gets a new content key but keeps the id, so clients add ?v=).
func (h *Handler) File(c *gin.Context) {
	id := c.Param("id")
	if !profiles.IsUUID(id) {
		c.Status(http.StatusNotFound)
		return
	}
	variant := "full"
	if c.Query("size") == "thumb" {
		variant = "thumb"
	}
	f, err := h.service.Open(c.Request.Context(), c.GetString("userID"), id, variant)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			c.Status(http.StatusNotFound)
			return
		}
		writeError(c, "photos.file", err)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Header("Content-Disposition", "inline")
	c.DataFromReader(http.StatusOK, info.Size(), "image/jpeg", f, nil)
}
