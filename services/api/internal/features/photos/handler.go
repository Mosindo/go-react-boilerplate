package photos

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/media"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, ErrPhotoNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", "photo not found", operation, err)
	case errors.Is(err, ErrTooManyPhotos):
		httpx.Fail(c, http.StatusConflict, "photo_limit", err.Error(), operation, err)
	case errors.Is(err, ErrInvalidOrder):
		httpx.Fail(c, http.StatusBadRequest, "invalid_order", err.Error(), operation, err)
	case errors.Is(err, media.ErrUnsupportedType):
		httpx.Fail(c, http.StatusUnsupportedMediaType, "unsupported_type", "only JPEG, PNG or WebP images are accepted", operation, err)
	case errors.Is(err, media.ErrImageTooLarge):
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "image_too_large", "image is too large (10 MB max)", operation, err)
	case errors.Is(err, media.ErrImageTooSmall):
		httpx.Fail(c, http.StatusBadRequest, "image_too_small", "image must be at least 200x200 pixels", operation, err)
	case errors.Is(err, media.ErrInvalidImage):
		httpx.Fail(c, http.StatusBadRequest, "invalid_image", "file is not a valid image", operation, err)
	default:
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not process photo", operation, err)
	}
}

// uploadedFile extracts the "photo" multipart field with a hard body limit.
func uploadedFile(c *gin.Context) (multipartFile, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, media.MaxUploadBytes+64<<10)
	file, _, err := c.Request.FormFile("photo")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, media.ErrImageTooLarge
		}
		return nil, media.ErrInvalidImage
	}
	return file, nil
}

type multipartFile interface {
	Read(p []byte) (int, error)
	Close() error
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Upload(c *gin.Context) {
	file, err := uploadedFile(c)
	if err != nil {
		h.fail(c, "photos.upload.read", err)
		return
	}
	defer file.Close()
	list, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), file)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, PhotosResponse{Photos: list})
}

func (h *Handler) Replace(c *gin.Context) {
	photoID := strings.TrimSpace(c.Param("photoId"))
	if !httpx.IsUUID(photoID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_id", "invalid photo id", "photos.replace.validate", nil)
		return
	}
	file, err := uploadedFile(c)
	if err != nil {
		h.fail(c, "photos.replace.read", err)
		return
	}
	defer file.Close()
	list, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, file)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Delete(c *gin.Context) {
	photoID := strings.TrimSpace(c.Param("photoId"))
	if !httpx.IsUUID(photoID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_id", "invalid photo id", "photos.delete.validate", nil)
		return
	}
	list, err := h.service.Delete(c.Request.Context(), httpx.UserID(c), photoID)
	if err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "photos.reorder.bind", err)
		return
	}
	for _, id := range req.PhotoIDs {
		if !httpx.IsUUID(id) {
			httpx.Fail(c, http.StatusBadRequest, "invalid_order", "invalid photo id", "photos.reorder.validate", nil)
			return
		}
	}
	list, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

// Serve streams a photo behind a valid signed URL. It is not behind the
// bearer middleware because image components cannot send headers everywhere;
// the HMAC signature and its expiry are the access control.
func (h *Handler) Serve(c *gin.Context) {
	photoID := strings.TrimSpace(c.Param("photoId"))
	if !httpx.IsUUID(photoID) {
		c.Status(http.StatusNotFound)
		return
	}
	path, err := h.service.Open(c.Request.Context(), photoID, c.Query("exp"), c.Query("sig"))
	if err != nil {
		if errors.Is(err, ErrPhotoNotFound) {
			c.Status(http.StatusNotFound)
			return
		}
		h.fail(c, "photos.serve", err)
		return
	}
	if _, err := os.Stat(path); err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	headers := c.Writer.Header()
	headers.Set("Content-Type", "image/jpeg")
	headers.Set("Cache-Control", "private, max-age=3600")
	headers.Set("Cross-Origin-Resource-Policy", "cross-origin")
	headers.Set("Content-Security-Policy", "default-src 'none'")
	c.File(path)
}
