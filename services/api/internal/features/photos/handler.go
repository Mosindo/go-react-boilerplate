package photos

import (
	"io"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// readUpload reads the multipart `file` field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+(1<<20))
	fh, err := c.FormFile("file")
	if err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_image", ErrInvalidImage.Message)
		return nil, false
	}
	if fh.Size > maxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "invalid_image", ErrInvalidImage.Message)
		return nil, false
	}
	f, err := fh.Open()
	if err != nil {
		httpx.BadRequest(c, "could not read upload")
		return nil, false
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxUploadBytes+1))
	if err != nil {
		httpx.BadRequest(c, "could not read upload")
		return nil, false
	}
	return raw, true
}

func (h *Handler) Upload(c *gin.Context) {
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	p, err := h.service.Add(c.Request.Context(), httpx.UserID(c), raw)
	if err != nil {
		httpx.Respond(c, "photos.add", err)
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) Replace(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	p, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), id, raw)
	if err != nil {
		httpx.Respond(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Respond(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "photoIds is required")
		return
	}
	photos, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		httpx.Respond(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

func (h *Handler) Content(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	data, contentType, err := h.service.Content(c.Request.Context(), id, c.Query("exp"), c.Query("sig"))
	if err != nil {
		httpx.Respond(c, "photos.content", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	c.Data(http.StatusOK, contentType, data)
}
