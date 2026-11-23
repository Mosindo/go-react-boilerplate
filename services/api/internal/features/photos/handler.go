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

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, ErrUnsupportedType):
		httpx.Fail(c, http.StatusUnsupportedMediaType, "unsupported_type", err.Error(), operation, nil)
	case errors.Is(err, ErrImageTooLarge):
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "image_too_large", "image must be at most 10 MB", operation, nil)
	case errors.Is(err, ErrImageTooSmall):
		httpx.BadRequest(c, "image_too_small", err.Error())
	case errors.Is(err, ErrInvalidImage):
		httpx.BadRequest(c, "invalid_image", err.Error())
	case errors.Is(err, ErrTooManyPhotos):
		httpx.Fail(c, http.StatusConflict, "too_many_photos", err.Error(), operation, nil)
	case errors.Is(err, ErrInvalidOrdering):
		httpx.BadRequest(c, "invalid_ordering", err.Error())
	case errors.Is(err, ErrPhotoNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", "photo not found", operation, nil)
	default:
		httpx.Internal(c, operation, err)
	}
}

// readUpload reads the multipart "photo" field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+64<<10)
	file, _, err := c.Request.FormFile("photo")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, ErrImageTooLarge
		}
		return nil, ErrInvalidImage
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, MaxUploadBytes+1))
	if err != nil {
		return nil, ErrInvalidImage
	}
	return data, nil
}

func (h *Handler) List(c *gin.Context) {
	res, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Upload(c *gin.Context) {
	data, err := readUpload(c)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	res, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), data)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	data, err := readUpload(c)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	res, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, data)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	res, err := h.service.Delete(c.Request.Context(), httpx.UserID(c), photoID)
	if err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "photoIds required")
		return
	}
	for _, id := range req.PhotoIDs {
		if !httpx.IsUUID(id) {
			httpx.BadRequest(c, "invalid_ordering", ErrInvalidOrdering.Error())
			return
		}
	}
	res, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// Serve streams a photo for a valid signed URL.
func (h *Handler) Serve(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	obj, err := h.service.Open(c.Request.Context(), photoID, c.Query("exp"), c.Query("sig"))
	if err != nil {
		h.fail(c, "photos.serve", err)
		return
	}
	defer obj.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=3600, immutable")
	c.Header("Content-Security-Policy", "default-src 'none'")
	c.Header("Cross-Origin-Resource-Policy", "cross-origin")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, obj)
}
