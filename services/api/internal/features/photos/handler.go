package photos

import (
	"errors"
	"io"
	"net/http"
	"time"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/imaging"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status, code, msg := http.StatusInternalServerError, "internal", "something went wrong"
	switch {
	case errors.Is(err, ErrNotFound):
		status, code, msg = http.StatusNotFound, "not_found", "photo not found"
	case errors.Is(err, ErrLimitReached):
		status, code, msg = http.StatusConflict, "photo_limit", "you can upload up to 6 photos"
	case errors.Is(err, ErrProfileRequired):
		status, code, msg = http.StatusConflict, "profile_required", "create your profile first"
	case errors.Is(err, ErrInvalidOrder):
		status, code, msg = http.StatusBadRequest, "invalid_order", err.Error()
	case errors.Is(err, imaging.ErrUnsupportedType), errors.Is(err, imaging.ErrTooSmall),
		errors.Is(err, imaging.ErrTooLarge), errors.Is(err, imaging.ErrCorrupt):
		status, code, msg = http.StatusBadRequest, "invalid_image", err.Error()
	}
	logger.LogHandlerError(c, op, status, err)
	httpx.Fail(c, status, code, msg)
}

// readUpload reads the "photo" multipart field with a hard size cap.
func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, imaging.MaxUploadBytes+64<<10)
	file, _, err := c.Request.FormFile("photo")
	if err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_image", "a 'photo' file of at most 10 MB is required")
		return nil, false
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, imaging.MaxUploadBytes+1))
	if err != nil || len(raw) > imaging.MaxUploadBytes {
		httpx.Fail(c, http.StatusBadRequest, "invalid_image", "image must be at most 10 MB")
		return nil, false
	}
	return raw, true
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "photos.list", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: items})
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
	c.JSON(http.StatusCreated, photo)
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
	photo, err := h.service.Replace(c.Request.Context(), httpx.UserID(c), photoID, raw)
	if err != nil {
		h.fail(c, "photos.replace", err)
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
		h.fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Reorder(c *gin.Context) {
	var req ReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "photoIds must be a list of photo ids")
		return
	}
	items, err := h.service.Reorder(c.Request.Context(), httpx.UserID(c), req.PhotoIDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, PhotosResponse{Photos: items})
}

// Serve streams a photo for a valid signed URL. The response is private
// (browser cache only) and served with a fixed, safe content type.
func (h *Handler) Serve(c *gin.Context) {
	photoID, ok := httpx.UUIDParam(c, "photoId")
	if !ok {
		return
	}
	f, err := h.service.Open(c.Request.Context(), photoID, c.Query("exp"), c.Query("sig"))
	if err != nil {
		httpx.Fail(c, http.StatusNotFound, "not_found", "photo not found")
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=3600, immutable")
	c.Header("Content-Disposition", "inline")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
