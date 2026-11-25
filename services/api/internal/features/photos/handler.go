package photos

import (
	"errors"
	"io"
	"net/http"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

var ErrMissingFile = apperr.Validation("missing_file", "a photo file is required in the `photo` field")

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Upload(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photo, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), data)
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photo, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, data)
	if err != nil {
		httpx.Fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, photo)
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), photoID); err != nil {
		httpx.Fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if !httpx.BindJSON(c, "photos.reorder", &req) {
		return
	}
	list, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		httpx.Fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

// Serve streams a photo file. Access is granted by the signed URL that the
// API only hands out to users allowed to see the profile.
func (h *Handler) Serve(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	path, err := h.service.FilePath(c.Request.Context(), photoID, c.Query("exp"), c.Query("sig"))
	if err != nil {
		httpx.Fail(c, "photos.serve", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	c.File(path)
}

func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+64<<10)
	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.Fail(c, "photos.read", ErrImageTooLarge)
			return nil, false
		}
		httpx.Fail(c, "photos.read", ErrMissingFile)
		return nil, false
	}
	defer file.Close()
	if header.Size > MaxUploadBytes {
		httpx.Fail(c, "photos.read", ErrImageTooLarge)
		return nil, false
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxUploadBytes+1))
	if err != nil || len(data) > MaxUploadBytes {
		httpx.Fail(c, "photos.read", ErrImageTooLarge)
		return nil, false
	}
	return data, true
}
