package photos

import (
	"errors"
	"io"
	"net/http"

	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/validate"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func toResponse(p Photo) PhotoResponse {
	return PhotoResponse{ID: p.ID, URL: "/photos/" + p.ID, Position: p.Position}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrUnsupportedType):
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": err.Error()})
	case errors.Is(err, ErrInvalidImage), errors.Is(err, ErrImageTooLarge), errors.Is(err, ErrBadOrder):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrTooBig):
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": err.Error()})
	case errors.Is(err, ErrLimit):
		c.JSON(http.StatusConflict, gin.H{"error": "you can upload up to 6 photos"})
	case errors.Is(err, ErrProfileNeeded):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "photo not found"})
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (h *Handler) upload(c *gin.Context, replaceID string) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+(1<<20))
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			h.fail(c, "photos.upload", ErrTooBig)
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "multipart field \"file\" is required"})
		return
	}
	defer file.Close()

	photo, err := h.service.Upload(c.Request.Context(), c.GetString("userID"), replaceID, file)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, toResponse(photo))
}

func (h *Handler) Create(c *gin.Context) { h.upload(c, "") }

func (h *Handler) Replace(c *gin.Context) {
	id := c.Param("photoId")
	if !validate.IsUUID(id) {
		h.fail(c, "photos.replace", ErrNotFound)
		return
	}
	h.upload(c, id)
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	out := make([]PhotoResponse, 0, len(items))
	for _, p := range items {
		out = append(out, toResponse(p))
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: out})
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("photoId")
	if !validate.IsUUID(id) {
		h.fail(c, "photos.delete", ErrNotFound)
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	seen := make(map[string]struct{}, len(req.IDs))
	for _, id := range req.IDs {
		if !validate.IsUUID(id) {
			h.fail(c, "photos.reorder", ErrBadOrder)
			return
		}
		if _, dup := seen[id]; dup {
			h.fail(c, "photos.reorder", ErrBadOrder)
			return
		}
		seen[id] = struct{}{}
	}
	if err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.IDs); err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	h.List(c)
}

// File streams the picture bytes after the access check in the repository.
func (h *Handler) File(c *gin.Context) {
	id := c.Param("photoId")
	if !validate.IsUUID(id) {
		h.fail(c, "photos.file", ErrNotFound)
		return
	}
	rc, err := h.service.Open(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		h.fail(c, "photos.file", err)
		return
	}
	defer rc.Close()
	c.Header("Content-Type", outputContentType)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Disposition", "inline")
	// A photo id is immutable (replacing creates a new id), so it can be cached by the client.
	c.Header("Cache-Control", "private, max-age=31536000, immutable")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, rc)
}
