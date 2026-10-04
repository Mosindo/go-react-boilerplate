package photos

import (
	"errors"
	"io"
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service  *Service
	maxBytes int64
}

func NewHandler(service *Service, maxBytes int64) *Handler {
	return &Handler{service: service, maxBytes: maxBytes}
}

func (h *Handler) Upload(c *gin.Context) {
	// Hard cap on the whole request before any parsing; +64KB for multipart framing.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBytes+64<<10)
	fh, err := c.FormFile("file")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httpx.Fail(c, "photos.upload", platformerrors.TooLarge("The image is too large"))
			return
		}
		httpx.Fail(c, "photos.upload", platformerrors.FieldInvalid("file", "Attach an image in the 'file' field"))
		return
	}
	if fh.Size > h.maxBytes {
		httpx.Fail(c, "photos.upload", platformerrors.TooLarge("The image is too large"))
		return
	}
	f, err := fh.Open()
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, h.maxBytes+1))
	if err != nil || int64(len(data)) > h.maxBytes {
		httpx.Fail(c, "photos.upload", platformerrors.TooLarge("The image is too large"))
		return
	}

	photo, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), data)
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Delete(c *gin.Context) {
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), c.Param("id")); err != nil {
		httpx.Fail(c, "photos.delete", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req OrderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	photos, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.IDs)
	if err != nil {
		httpx.Fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

func (h *Handler) File(c *gin.Context) {
	file, err := h.service.Open(c.Request.Context(), httpx.UserID(c), c.Param("id"), c.Query("size"))
	if err != nil {
		httpx.Fail(c, "photos.file", err)
		return
	}
	defer file.Content.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	http.ServeContent(c.Writer, c.Request, "", file.UpdatedAt, file.Content)
}
