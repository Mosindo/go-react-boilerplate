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

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "photo not found")
	case errors.Is(err, ErrTooMany):
		httpx.Error(c, http.StatusConflict, "photo limit reached")
	case errors.Is(err, ErrNeedProfile):
		httpx.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrUnsupportedType):
		httpx.Error(c, http.StatusUnsupportedMediaType, err.Error())
	case errors.Is(err, ErrInvalidImage), errors.Is(err, ErrInvalidOrder):
		httpx.BadRequest(c, err.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadSize+(64<<10))
	fh, err := c.FormFile("photo")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Error(c, http.StatusRequestEntityTooLarge, "photo exceeds 8 MB")
			return nil, false
		}
		httpx.BadRequest(c, "photo file is required")
		return nil, false
	}
	if fh.Size > MaxUploadSize {
		httpx.Error(c, http.StatusRequestEntityTooLarge, "photo exceeds 8 MB")
		return nil, false
	}
	f, err := fh.Open()
	if err != nil {
		httpx.BadRequest(c, "unreadable upload")
		return nil, false
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, MaxUploadSize+1))
	if err != nil {
		httpx.BadRequest(c, "unreadable upload")
		return nil, false
	}
	return raw, true
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": items})
}

func (h *Handler) Upload(c *gin.Context) {
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	p, err := h.service.Add(c.Request.Context(), httpx.UserID(c), raw)
	if err != nil {
		fail(c, "photos.add", err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) Replace(c *gin.Context) {
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	p, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), c.Param("id"), raw)
	if err != nil {
		fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req OrderRequest
	if !httpx.Bind(c, &req) {
		return
	}
	items, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.IDs)
	if err != nil {
		fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": items})
}

func (h *Handler) Content(c *gin.Context) {
	content, err := h.service.Content(c.Request.Context(), httpx.UserID(c), c.Param("id"))
	if err != nil {
		fail(c, "photos.content", err)
		return
	}
	sum := sha256.Sum256(content.Data)
	etag := `"` + hex.EncodeToString(sum[:8]) + `"`
	c.Header("ETag", etag)
	c.Header("Cache-Control", "private, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Data(http.StatusOK, content.ContentType, content.Data)
}
