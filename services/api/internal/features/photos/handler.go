package photos

import (
	"errors"
	"io"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/media"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrProfileRequired), errors.Is(err, ErrTooManyPhotos), errors.Is(err, ErrInvalidOrder):
		httpx.Fail(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrPhotoNotFound):
		httpx.Fail(c, http.StatusNotFound, "photo not found")
	case errors.Is(err, media.ErrUnsupportedType):
		httpx.Fail(c, http.StatusUnsupportedMediaType, "only JPEG, PNG or WebP images are accepted")
	case errors.Is(err, media.ErrImageTooSmall):
		httpx.Fail(c, http.StatusBadRequest, "image is too small (200px minimum)")
	case errors.Is(err, media.ErrImageTooLarge):
		httpx.Fail(c, http.StatusBadRequest, "image dimensions are too large")
	case errors.Is(err, media.ErrCorruptImage):
		httpx.Fail(c, http.StatusBadRequest, "image could not be read")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Fail(c, http.StatusInternalServerError, "could not process photo")
	}
}

// readUpload extracts the "photo" multipart field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, media.MaxUploadBytes+64*1024)
	file, _, err := c.Request.FormFile("photo")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.Fail(c, http.StatusRequestEntityTooLarge, "photo is too large (10 MB max)")
			return nil, false
		}
		httpx.Fail(c, http.StatusBadRequest, "multipart field \"photo\" is required")
		return nil, false
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, media.MaxUploadBytes+1))
	if err != nil {
		httpx.Fail(c, http.StatusBadRequest, "could not read upload")
		return nil, false
	}
	if len(raw) > media.MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "photo is too large (10 MB max)")
		return nil, false
	}
	return raw, true
}

func (h *Handler) respond(c *gin.Context, op string, status int, list []Photo, err error) {
	if err != nil {
		h.fail(c, op, err)
		return
	}
	c.JSON(status, PhotosResponse{Photos: list})
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	h.respond(c, "photos.list", http.StatusOK, list, err)
}

func (h *Handler) Upload(c *gin.Context) {
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	list, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), raw)
	h.respond(c, "photos.upload", http.StatusCreated, list, err)
}

func (h *Handler) Replace(c *gin.Context) {
	photoID := c.Param("photoId")
	if !httpx.IsUUID(photoID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid photo id")
		return
	}
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	list, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, raw)
	h.respond(c, "photos.replace", http.StatusOK, list, err)
}

func (h *Handler) Delete(c *gin.Context) {
	photoID := c.Param("photoId")
	if !httpx.IsUUID(photoID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid photo id")
		return
	}
	list, err := h.service.Delete(c.Request.Context(), httpx.UserID(c), photoID)
	h.respond(c, "photos.delete", http.StatusOK, list, err)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}
	for _, id := range req.PhotoIDs {
		if !httpx.IsUUID(id) {
			httpx.Fail(c, http.StatusBadRequest, "invalid photo id")
			return
		}
	}
	list, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	h.respond(c, "photos.reorder", http.StatusOK, list, err)
}
