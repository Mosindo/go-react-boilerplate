package photos

import (
	"errors"
	"io"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func readUpload(c *gin.Context) ([]byte, bool) {
	fileHeader, err := c.FormFile("photo")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httpx.Fail(c, "photos.upload", ErrImageTooLarge)
			return nil, false
		}
		httpx.Abort(c, http.StatusBadRequest, "photo_required", "multipart field \"photo\" is required")
		return nil, false
	}
	if fileHeader.Size > MaxUploadBytes {
		httpx.Fail(c, "photos.upload", ErrImageTooLarge)
		return nil, false
	}
	file, err := fileHeader.Open()
	if err != nil {
		httpx.Fail(c, "photos.upload.open", err)
		return nil, false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxUploadBytes+1))
	if err != nil {
		httpx.Fail(c, "photos.upload.read", err)
		return nil, false
	}
	return data, true
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.ListMine(c.Request.Context(), httpx.UserID(c))
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

// Serve streams a photo behind a signed, expiring URL.
func (h *Handler) Serve(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	file, photo, err := h.service.OpenSigned(c.Request.Context(), photoID, c.Query("exp"), c.Query("sig"))
	if err != nil {
		httpx.Fail(c, "photos.serve", err)
		return
	}
	defer file.Close()
	headers := c.Writer.Header()
	headers.Set("Content-Type", "image/jpeg")
	headers.Set("Cache-Control", "private, max-age=3600")
	headers.Set("Cross-Origin-Resource-Policy", "cross-origin")
	headers.Set("Content-Security-Policy", "default-src 'none'")
	http.ServeContent(c.Writer, c.Request, "", photo.CreatedAt.Truncate(time.Second), file)
}
