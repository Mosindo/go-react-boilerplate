package photos

import (
	"errors"
	"io"
	"net/http"
	"regexp"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		logger.LogHandlerError(c, "photos.list", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: toResponses(items)})
}

// Upload accepts multipart/form-data with a "file" part. Optional ?replace=<photoId>.
func (h *Handler) Upload(c *gin.Context) {
	replaceID := c.Query("replace")
	if replaceID != "" && !uuidPattern.MatchString(replaceID) {
		httpx.Error(c, http.StatusBadRequest, "invalid replace id")
		return
	}

	// Hard cap on what is read from the socket, whatever the client claims.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBytes+(1<<20))
	header, err := c.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Error(c, http.StatusRequestEntityTooLarge, "file too large (max 8 MB)")
			return
		}
		httpx.Error(c, http.StatusBadRequest, "missing file")
		return
	}
	if header.Size > maxUploadBytes {
		httpx.Error(c, http.StatusRequestEntityTooLarge, "file too large (max 8 MB)")
		return
	}
	file, err := header.Open()
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "unreadable file")
		return
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil || len(raw) > maxUploadBytes {
		httpx.Error(c, http.StatusRequestEntityTooLarge, "file too large (max 8 MB)")
		return
	}

	photo, err := h.service.Upload(c.Request.Context(), c.GetString("userID"), raw, replaceID)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnsupportedType):
			httpx.Error(c, http.StatusUnsupportedMediaType, err.Error())
		case errors.Is(err, ErrImageTooLarge), errors.Is(err, ErrImageTooSmall), errors.Is(err, ErrCorruptImage):
			httpx.Error(c, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, ErrLimitReached):
			httpx.Error(c, http.StatusConflict, "photo limit reached (max 6)")
		case errors.Is(err, ErrNotFound):
			httpx.Error(c, http.StatusNotFound, "photo not found")
		default:
			logger.LogHandlerError(c, "photos.upload", http.StatusInternalServerError, err)
			httpx.Internal(c)
		}
		return
	}
	c.JSON(http.StatusCreated, toResponse(photo))
}

func (h *Handler) Delete(c *gin.Context) {
	id := c.Param("photoId")
	if !uuidPattern.MatchString(id) {
		httpx.Error(c, http.StatusNotFound, "photo not found")
		return
	}
	if err := h.service.Delete(c.Request.Context(), c.GetString("userID"), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, "photo not found")
			return
		}
		logger.LogHandlerError(c, "photos.delete", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}

// Reorder sets the display order; the first id becomes the primary photo.
func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	for _, id := range req.PhotoIDs {
		if !uuidPattern.MatchString(id) {
			httpx.Error(c, http.StatusBadRequest, "invalid photo id")
			return
		}
	}
	userID := c.GetString("userID")
	if err := h.service.Reorder(c.Request.Context(), userID, req.PhotoIDs); err != nil {
		if errors.Is(err, ErrInvalidOrder) {
			httpx.Error(c, http.StatusUnprocessableEntity, err.Error())
			return
		}
		logger.LogHandlerError(c, "photos.reorder", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	h.List(c)
}

// Content streams an image. Photos are private: every read is authorised and
// answers 404 when the viewer is not allowed to see the owner.
func (h *Handler) Content(c *gin.Context) {
	id := c.Param("photoId")
	if !uuidPattern.MatchString(id) {
		httpx.Error(c, http.StatusNotFound, "photo not found")
		return
	}
	photo, f, err := h.service.Open(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(c, http.StatusNotFound, "photo not found")
			return
		}
		logger.LogHandlerError(c, "photos.content", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	defer f.Close()

	// Photo ids are never reused and their bytes never change, so clients may cache forever.
	c.Header("Cache-Control", "private, max-age=31536000, immutable")
	c.Header("Content-Type", photo.ContentType)
	c.Header("Content-Disposition", "inline")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}

func toResponses(items []Photo) []PhotoResponse {
	out := make([]PhotoResponse, 0, len(items))
	for _, p := range items {
		out = append(out, toResponse(p))
	}
	return out
}
