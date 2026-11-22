package photos

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

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

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

// readUpload reads the "photo" multipart field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, error) {
	file, _, err := c.Request.FormFile("photo")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, media.ErrImageTooLarge
		}
		return nil, ErrMissingPayload
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, media.MaxUploadBytes+1))
	if err != nil {
		return nil, ErrMissingPayload
	}
	if len(data) > media.MaxUploadBytes {
		return nil, media.ErrImageTooLarge
	}
	return data, nil
}

func (h *Handler) Upload(c *gin.Context) {
	data, err := readUpload(c)
	if err != nil {
		h.fail(c, "photos.upload.read", err)
		return
	}
	list, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), data)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, PhotosResponse{Photos: list})
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := photoParam(c)
	if !ok {
		return
	}
	data, err := readUpload(c)
	if err != nil {
		h.fail(c, "photos.replace.read", err)
		return
	}
	list, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, data)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := photoParam(c)
	if !ok {
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
		httpx.BadRequest(c, "invalid_order", ErrInvalidOrder.Error())
		return
	}
	list, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: list})
}

// Serve streams a photo for a valid signed URL. No bearer token is needed so
// that <img>/<Image> tags can load it, but the URL is unguessable,
// viewer-scoped at issue time and expiring.
func (h *Handler) Serve(c *gin.Context) {
	photoID := strings.TrimSpace(c.Param("photoId"))
	if !httpx.IsUUID(photoID) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	f, expires, err := h.service.Open(c.Request.Context(), photoID, c.Query("exp"), c.Query("sig"))
	if err != nil {
		if errors.Is(err, ErrPhotoNotFound) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		httpx.Internal(c, "photos.serve", err)
		return
	}
	defer f.Close()
	maxAge := int(time.Until(expires).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	c.Header("Cache-Control", "private, max-age="+strconv.Itoa(maxAge)+", immutable")
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	c.Header("Content-Security-Policy", "default-src 'none'")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}

func photoParam(c *gin.Context) (string, bool) {
	id := strings.TrimSpace(c.Param("photoId"))
	if !httpx.IsUUID(id) {
		httpx.BadRequest(c, "invalid_photo_id", "invalid photo id")
		return "", false
	}
	return id, true
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrMissingPayload):
		httpx.Fail(c, op, http.StatusBadRequest, "photo_required", "a photo file is required (multipart field \"photo\")", nil)
	case errors.Is(err, media.ErrUnsupportedType):
		httpx.Fail(c, op, http.StatusUnsupportedMediaType, "unsupported_type", "only JPEG and PNG images are accepted", nil)
	case errors.Is(err, media.ErrImageTooLarge):
		httpx.Fail(c, op, http.StatusRequestEntityTooLarge, "image_too_large", "image is too large (10 MB / 40 MP max)", nil)
	case errors.Is(err, media.ErrImageTooSmall):
		httpx.Fail(c, op, http.StatusBadRequest, "image_too_small", "image must be at least 200×200 pixels", nil)
	case errors.Is(err, media.ErrCorruptImage):
		httpx.Fail(c, op, http.StatusBadRequest, "corrupt_image", "image could not be read", nil)
	case errors.Is(err, ErrTooManyPhotos):
		httpx.Fail(c, op, http.StatusConflict, "photo_limit", "you can have up to 6 photos", nil)
	case errors.Is(err, ErrInvalidOrder):
		httpx.Fail(c, op, http.StatusBadRequest, "invalid_order", err.Error(), nil)
	case errors.Is(err, ErrPhotoNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "photo_not_found", "photo not found", nil)
	default:
		httpx.Internal(c, op, err)
	}
}
