package photos

import (
	"errors"
	"io"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

type orderRequest struct {
	IDs []string `json:"ids" binding:"required,max=6"`
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrUnsupportedImage):
		httpx.Fail(c, http.StatusUnsupportedMediaType, err.Error())
	case errors.Is(err, ErrImageTooLarge):
		httpx.Fail(c, http.StatusRequestEntityTooLarge, err.Error())
	case errors.Is(err, ErrLimit):
		httpx.Fail(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "photo not found")
	case errors.Is(err, ErrInvalidOrder):
		httpx.Fail(c, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, ErrBadSignature):
		httpx.Fail(c, http.StatusForbidden, err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}

// readUpload reads the multipart "file" field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+(64<<10))
	fh, err := c.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Fail(c, http.StatusRequestEntityTooLarge, "file is larger than 8 MB")
		} else {
			httpx.Fail(c, http.StatusBadRequest, "missing file")
		}
		return nil, false
	}
	if fh.Size > maxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "file is larger than 8 MB")
		return nil, false
	}
	f, err := fh.Open()
	if err != nil {
		httpx.Fail(c, http.StatusBadRequest, "unreadable file")
		return nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxUploadBytes+1))
	if err != nil || len(data) > maxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "file is larger than 8 MB")
		return nil, false
	}
	return data, true
}

func (h *Handler) Upload(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	id, err := h.service.Upload(c.Request.Context(), c.GetString("userID"), data)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handler) Replace(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	data, ok := readUpload(c)
	if !ok {
		return
	}
	if err := h.service.Replace(c.Request.Context(), c.GetString("userID"), id, data); err != nil {
		h.fail(c, "photos.replace", err)
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

func (h *Handler) Reorder(c *gin.Context) {
	var req orderRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	for _, id := range req.IDs {
		if !httpx.IsUUID(id) {
			httpx.Fail(c, http.StatusUnprocessableEntity, "invalid photo id")
			return
		}
	}
	if err := h.service.Reorder(c.Request.Context(), c.GetString("userID"), req.IDs); err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Serve streams a photo for a valid signed link. No bearer token is needed so
// that standard image components can load it; the signature is the capability.
func (h *Handler) Serve(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	size := c.DefaultQuery("size", "full")
	data, err := h.service.Open(id, size, c.Query("exp"), c.Query("sig"))
	if err != nil {
		h.fail(c, "photos.serve", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=21600")
	c.Header("Content-Disposition", "inline")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Data(http.StatusOK, "image/jpeg", data)
}
