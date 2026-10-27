package profiles

import (
	"errors"
	"io"
	"net/http"

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
	var verr ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.Error(c, http.StatusBadRequest, verr.Message)
	case errors.Is(err, ErrUnderage), errors.Is(err, ErrUnknownInterest), errors.Is(err, ErrBadPhotoOrder), errors.Is(err, ErrInvalidImage):
		httpx.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "profile not found")
	case errors.Is(err, ErrProfileRequired):
		httpx.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrPhotoNotFound):
		httpx.Error(c, http.StatusNotFound, "photo not found")
	case errors.Is(err, ErrPhotoLimit):
		httpx.Error(c, http.StatusConflict, "photo limit reached")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) GetMine(c *gin.Context) {
	p, err := h.service.Get(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profiles.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) PutMine(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.Upsert(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profiles.upsert", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) PutPreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.UpdatePreferences(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) PutPrivacy(c *gin.Context) {
	var req UpdatePrivacyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.UpdatePrivacy(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profiles.privacy", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) ListInterests(c *gin.Context) {
	items, err := h.service.Interests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": items})
}

func readUpload(c *gin.Context) ([]byte, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, imaging.MaxUploadBytes+(1<<20))
	file, err := c.FormFile("photo")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			httpx.Error(c, http.StatusRequestEntityTooLarge, "picture is too large (max 8 MB)")
			return nil, false
		}
		httpx.Error(c, http.StatusBadRequest, "a 'photo' file is required (max 8 MB)")
		return nil, false
	}
	if file.Size > imaging.MaxUploadBytes {
		httpx.Error(c, http.StatusRequestEntityTooLarge, "picture is too large (max 8 MB)")
		return nil, false
	}
	f, err := file.Open()
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "could not read file")
		return nil, false
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, imaging.MaxUploadBytes+1))
	if err != nil || len(data) > imaging.MaxUploadBytes {
		httpx.Error(c, http.StatusBadRequest, "could not read file")
		return nil, false
	}
	return data, true
}

func (h *Handler) AddPhoto(c *gin.Context) {
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photo, err := h.service.AddPhoto(c.Request.Context(), c.GetString("userID"), data)
	if err != nil {
		h.fail(c, "photos.add", err)
		return
	}
	c.JSON(http.StatusCreated, photo)
}

func (h *Handler) ReplacePhoto(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	data, ok := readUpload(c)
	if !ok {
		return
	}
	photo, err := h.service.ReplacePhoto(c.Request.Context(), c.GetString("userID"), id, data)
	if err != nil {
		h.fail(c, "photos.replace", err)
		return
	}
	c.JSON(http.StatusOK, photo)
}

func (h *Handler) DeletePhoto(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeletePhoto(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "photos.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ReorderPhotos(c *gin.Context) {
	var req ReorderPhotosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.ReorderPhotos(c.Request.Context(), c.GetString("userID"), req.IDs)
	if err != nil {
		h.fail(c, "photos.reorder", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) ServePhoto(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	photo, f, err := h.service.OpenPhoto(c.Request.Context(), id, c.GetString("userID"))
	if err != nil {
		h.fail(c, "photos.serve", err)
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("ETag", `"`+photo.StorageKey+`"`)
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, f)
}
