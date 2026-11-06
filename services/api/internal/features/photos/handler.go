package photos

import (
	"errors"
	"io"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "photo not found")
	case errors.Is(err, ErrLimitReached):
		httpx.Error(c, http.StatusConflict, "photo_limit", "you can have at most 6 photos")
	case errors.Is(err, ErrBadOrder):
		httpx.BadRequest(c, err.Error())
	case errors.Is(err, ErrUnsupportedType):
		httpx.Error(c, http.StatusUnsupportedMediaType, "unsupported_type", err.Error())
	case errors.Is(err, ErrCorruptImage), errors.Is(err, ErrImageTooLarge), errors.Is(err, ErrImageTooSmall):
		httpx.BadRequest(c, err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}

// readUpload reads the multipart "file" field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(64<<10))
	header, err := c.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Error(c, http.StatusRequestEntityTooLarge, "file_too_large", "photo must be at most 8 MB")
		} else {
			httpx.BadRequest(c, "send the photo as multipart field \"file\"")
		}
		return nil, false
	}
	if header.Size > MaxUploadBytes {
		httpx.Error(c, http.StatusRequestEntityTooLarge, "file_too_large", "photo must be at most 8 MB")
		return nil, false
	}
	f, err := header.Open()
	if err != nil {
		httpx.BadRequest(c, "could not read the uploaded file")
		return nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxUploadBytes+1))
	if err != nil || len(data) > MaxUploadBytes {
		httpx.Error(c, http.StatusRequestEntityTooLarge, "file_too_large", "photo must be at most 8 MB")
		return nil, false
	}
	return data, true
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, ListResponse{Photos: items})
}

func (h *Handler) Upload(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	p, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), data)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) Replace(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	p, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), c.Param("photoId"), data)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), c.Param("photoId")); err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "ids is required")
		return
	}
	if err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.IDs); err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	h.List(c)
}

// Serve streams a photo to an authenticated viewer who is allowed to see it.
func (h *Handler) Serve(c *gin.Context) {
	f, key, err := h.service.Open(c.Request.Context(), httpx.UserID(c), c.Param("photoId"))
	if err != nil {
		h.fail(c, "photos.serve", err)
		return
	}
	defer f.Close()
	c.Header("Content-Type", storedMime)
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	c.Header("ETag", `"`+key+`"`)
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
