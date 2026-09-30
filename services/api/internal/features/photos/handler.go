package photos

import (
	"errors"
	"io"
	"net/http"

	apperr "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	photos, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: photos})
}

func (h *Handler) Upload(c *gin.Context) {
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	photos, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), raw)
	if err != nil {
		httpx.Fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, PhotosResponse{Photos: photos})
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	photos, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, raw)
	if err != nil {
		httpx.Fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: photos})
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	photos, err := h.service.Delete(c.Request.Context(), httpx.UserID(c), photoID)
	if err != nil {
		httpx.Fail(c, "photos.delete", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: photos})
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "photos.reorder.bind", err)
		return
	}
	photos, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		httpx.Fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: photos})
}

// Serve streams a photo for a valid signed URL.
func (h *Handler) Serve(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	file, err := h.service.Open(c.Request.Context(), photoID, c.Query("exp"), c.Query("sig"))
	if err != nil {
		httpx.Fail(c, "photos.serve", err)
		return
	}
	defer file.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=3600, immutable")
	c.Header("Content-Security-Policy", "default-src 'none'")
	// Signed media URLs are embedded by the app (native or web), so allow
	// cross-origin loading of this resource only.
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	http.ServeContent(c.Writer, c.Request, "photo.jpg", zeroTime, file)
}

func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+64*1024)
	fileHeader, err := c.FormFile("photo")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httpx.Fail(c, "photos.read", ErrImageTooLarge)
			return nil, false
		}
		httpx.Fail(c, "photos.read", apperr.Validation("multipart field 'photo' is required"))
		return nil, false
	}
	if fileHeader.Size > MaxUploadBytes {
		httpx.Fail(c, "photos.read", ErrImageTooLarge)
		return nil, false
	}
	f, err := fileHeader.Open()
	if err != nil {
		httpx.Fail(c, "photos.read", err)
		return nil, false
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, MaxUploadBytes+1))
	if err != nil {
		httpx.Fail(c, "photos.read", err)
		return nil, false
	}
	return raw, true
}
