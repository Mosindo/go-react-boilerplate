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
	case errors.Is(err, ErrUnsupportedType), errors.Is(err, ErrImageTooSmall), errors.Is(err, ErrImageTooLarge),
		errors.Is(err, ErrCorruptImage), errors.Is(err, ErrInvalidOrdering):
		httpx.FailCode(c, operation, http.StatusBadRequest, "validation", err.Error(), err)
	case errors.Is(err, ErrFileTooLarge):
		httpx.FailCode(c, operation, http.StatusRequestEntityTooLarge, "file_too_large", err.Error(), err)
	case errors.Is(err, ErrTooManyPhotos):
		httpx.FailCode(c, operation, http.StatusConflict, "too_many_photos", err.Error(), err)
	case errors.Is(err, ErrPhotoNotFound):
		httpx.Fail(c, operation, http.StatusNotFound, "photo not found", err)
	default:
		httpx.Fail(c, operation, http.StatusInternalServerError, "could not process photo", err)
	}
}

// uploadReader extracts the "photo" multipart field with a hard size cap.
func uploadReader(c *gin.Context) (io.ReadCloser, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+64<<10)
	file, _, err := c.Request.FormFile("photo")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, ErrFileTooLarge
		}
		return nil, err
	}
	return file, nil
}

func (h *Handler) List(c *gin.Context) {
	photos, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

func (h *Handler) Upload(c *gin.Context) {
	file, err := uploadReader(c)
	if err != nil {
		if errors.Is(err, ErrFileTooLarge) {
			h.fail(c, "photos.upload.read", err)
			return
		}
		httpx.BadRequest(c, "photos.upload.read", "send the image in a multipart field named \"photo\"", err)
		return
	}
	defer file.Close()

	photo, err := h.service.Upload(c.Request.Context(), httpx.UserID(c), file)
	if err != nil {
		h.fail(c, "photos.upload", err)
		return
	}
	logger.LogHandlerEvent(c, "photos.upload.success", http.StatusCreated, map[string]string{"photo_id": photo.ID})
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) Replace(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId", "photos.replace.validate")
	if !ok {
		return
	}
	file, err := uploadReader(c)
	if err != nil {
		if errors.Is(err, ErrFileTooLarge) {
			h.fail(c, "photos.replace.read", err)
			return
		}
		httpx.BadRequest(c, "photos.replace.read", "send the image in a multipart field named \"photo\"", err)
		return
	}
	defer file.Close()

	photo, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, file)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, photo)
}

func (h *Handler) Delete(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId", "photos.delete.validate")
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
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "photos.reorder.bind", "photoIds must list your photo ids", err)
		return
	}
	photos, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

// Serve streams a photo for a valid signed URL. No bearer token is needed so
// that native image components and browsers can load it, but the URL is only
// handed to authorized viewers and expires within two hours.
func (h *Handler) Serve(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId", "photos.serve.validate")
	if !ok {
		return
	}
	rc, err := h.service.Open(c.Request.Context(), photoID, c.Query("exp"), c.Query("sig"))
	if err != nil {
		if errors.Is(err, ErrInvalidSignature) {
			httpx.Fail(c, "photos.serve", http.StatusForbidden, "link expired", nil)
			return
		}
		h.fail(c, "photos.serve", err)
		return
	}
	defer rc.Close()

	headers := c.Writer.Header()
	headers.Set("Cache-Control", "private, max-age=3600, immutable")
	headers.Set("Cross-Origin-Resource-Policy", "cross-origin")
	headers.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	c.DataFromReader(http.StatusOK, -1, "image/jpeg", rc, nil)
}
