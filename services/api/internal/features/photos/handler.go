package photos

import (
	"errors"
	"io"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
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
		httpx.Fail(c, http.StatusNotFound, "not_found", "photo not found")
	case errors.Is(err, ErrPhotoLimit):
		httpx.Fail(c, http.StatusConflict, "photo_limit", err.Error())
	case errors.Is(err, ErrLastPhoto):
		httpx.Fail(c, http.StatusConflict, "last_photo", err.Error())
	case errors.Is(err, ErrInvalidOrder):
		httpx.Fail(c, http.StatusBadRequest, "invalid_order", err.Error())
	case errors.Is(err, ErrUnsupportedType):
		httpx.Fail(c, http.StatusUnsupportedMediaType, "unsupported_type", err.Error())
	case errors.Is(err, ErrInvalidImage), errors.Is(err, ErrImageTooSmall), errors.Is(err, ErrImageTooLarge):
		httpx.Fail(c, http.StatusUnprocessableEntity, "invalid_image", err.Error())
	default:
		httpx.FailInternal(c, operation, err)
	}
}

// readUpload extracts the "photo" multipart field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+64<<10)
	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "photos must be 10 MB or less")
			return nil, false
		}
		logger.LogHandlerError(c, "photos.form", http.StatusBadRequest, err)
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "expected a multipart field named \"photo\"")
		return nil, false
	}
	defer file.Close()
	if header.Size > MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "photos must be 10 MB or less")
		return nil, false
	}
	raw, err := io.ReadAll(io.LimitReader(file, MaxUploadBytes+1))
	if err != nil || len(raw) == 0 {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "could not read the uploaded file")
		return nil, false
	}
	if len(raw) > MaxUploadBytes {
		httpx.Fail(c, http.StatusRequestEntityTooLarge, "file_too_large", "photos must be 10 MB or less")
		return nil, false
	}
	return raw, true
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, toResponses(list))
}

func (h *Handler) Upload(c *gin.Context) {
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	photo, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), raw)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	c.JSON(http.StatusCreated, toResponse(photo))
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.PathUUID(c, "photoId")
	if !ok {
		return
	}
	raw, ok := readUpload(c)
	if !ok {
		return
	}
	photo, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, raw)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, toResponse(photo))
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := httpx.PathUUID(c, "photoId")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), httpx.UserID(c), photoID); err != nil {
		h.fail(c, "photos.delete", err)
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
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, toResponses(list))
}

// Media serves a photo for a signed URL. The signature (not a bearer token)
// authorises the request so <Image> components can load it directly.
func (h *Handler) Media(c *gin.Context) {
	photoID, ok := httpx.PathUUID(c, "photoId")
	if !ok {
		return
	}
	rc, err := h.service.OpenSigned(c.Request.Context(), photoID, c.Query("exp"), c.Query("sig"))
	if err != nil {
		if errors.Is(err, ErrInvalidSignature) {
			httpx.Fail(c, http.StatusForbidden, "forbidden", err.Error())
			return
		}
		h.fail(c, "photos.media", err)
		return
	}
	defer rc.Close()
	headers := c.Writer.Header()
	headers.Set("Content-Type", "image/jpeg")
	headers.Set("Cache-Control", "private, max-age=21600, immutable")
	headers.Set("Cross-Origin-Resource-Policy", "cross-origin")
	headers.Set("Content-Disposition", "inline")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, rc)
}
