package photos

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/media"
	"github.com/gin-gonic/gin"
)

var ErrInvalidSignature = errors.New("invalid media signature")

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// readUpload reads the "photo" multipart field with a hard size limit.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(512<<10))
	file, _, err := c.Request.FormFile("photo")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			httpx.Error(c, http.StatusRequestEntityTooLarge, "file_too_large", ErrImageTooLarge.Error())
			return nil, false
		}
		httpx.Error(c, http.StatusBadRequest, "photo_required", "multipart field 'photo' is required")
		return nil, false
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxUploadBytes+1))
	if err != nil {
		httpx.Error(c, http.StatusRequestEntityTooLarge, "file_too_large", ErrImageTooLarge.Error())
		return nil, false
	}
	return data, true
}

func (h *Handler) List(c *gin.Context) {
	photos, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: photos})
}

func (h *Handler) Upload(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photos, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), data)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, PhotosResponse{Photos: photos})
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		httpx.Error(c, http.StatusBadRequest, "invalid_photo_id", "invalid photo id")
		return
	}
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photos, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, data)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: photos})
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		httpx.Error(c, http.StatusBadRequest, "invalid_photo_id", "invalid photo id")
		return
	}
	photos, err := h.service.Delete(c.Request.Context(), httpx.UserID(c), photoID)
	if err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: photos})
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.InvalidRequest(c)
		return
	}
	for i, id := range req.PhotoIDs {
		id = strings.ToLower(strings.TrimSpace(id))
		if !httpx.IsUUID(id) {
			httpx.Error(c, http.StatusBadRequest, "invalid_photo_id", "invalid photo id")
			return
		}
		req.PhotoIDs[i] = id
	}
	photos, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: photos})
}

// Serve streams a photo for a valid signed URL. No bearer token is required
// (image components cannot always send headers) but URLs are unguessable,
// bound to one photo and variant, and expire.
func (h *Handler) Serve(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	variant := media.Variant(c.DefaultQuery("v", string(media.VariantFull)))
	file, err := h.service.Open(c.Request.Context(), photoID, variant, c.Query("exp"), c.Query("sig"))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidSignature):
			c.Status(http.StatusForbidden)
		case errors.Is(err, ErrPhotoNotFound):
			c.Status(http.StatusNotFound)
		default:
			logger.LogHandlerError(c, "photos.serve", http.StatusInternalServerError, err)
			c.Status(http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()
	headers := c.Writer.Header()
	headers.Set("Content-Type", "image/jpeg")
	headers.Set("Cache-Control", "private, max-age=3600")
	headers.Set("Content-Disposition", "inline")
	headers.Set("Cross-Origin-Resource-Policy", "cross-origin")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, file)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, ErrUnsupportedType), errors.Is(err, ErrImageCorrupt), errors.Is(err, ErrImageTooSmall), errors.Is(err, ErrImageDimensions):
		status = http.StatusUnprocessableEntity
		httpx.Error(c, status, "invalid_image", err.Error())
	case errors.Is(err, ErrImageTooLarge):
		status = http.StatusRequestEntityTooLarge
		httpx.Error(c, status, "file_too_large", err.Error())
	case errors.Is(err, ErrPhotoLimit):
		status = http.StatusConflict
		httpx.Error(c, status, "photo_limit", "you can upload up to 6 photos")
	case errors.Is(err, ErrInvalidOrder):
		status = http.StatusBadRequest
		httpx.Error(c, status, "invalid_order", err.Error())
	case errors.Is(err, ErrPhotoNotFound):
		status = http.StatusNotFound
		httpx.Error(c, status, "photo_not_found", "photo not found")
	default:
		httpx.Internal(c, "could not process photo")
	}
	logger.LogHandlerError(c, op, status, err)
}
