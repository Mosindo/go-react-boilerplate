package profiles

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	platformerrors "example.com/api/internal/platform/errors"
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
	case errors.Is(err, ErrProfileNotFound):
		platformerrors.Respond(c, http.StatusNotFound, "profile_not_found", "profile not found")
	case errors.Is(err, ErrPhotoNotFound):
		platformerrors.Respond(c, http.StatusNotFound, "photo_not_found", "photo not found")
	case errors.Is(err, ErrUnderage):
		platformerrors.Respond(c, http.StatusUnprocessableEntity, "underage", "you must be at least 18 years old")
	case errors.Is(err, ErrInvalidInput):
		platformerrors.Respond(c, http.StatusBadRequest, "invalid_request", "invalid request")
	case errors.Is(err, ErrPhotoLimit):
		platformerrors.Respond(c, http.StatusConflict, "photo_limit", "photo limit reached")
	case errors.Is(err, ErrImageTooLarge):
		platformerrors.Respond(c, http.StatusRequestEntityTooLarge, "image_too_large", "image is too large (max 8 MB)")
	case errors.Is(err, ErrInvalidImage):
		platformerrors.Respond(c, http.StatusUnsupportedMediaType, "invalid_image", "unsupported or invalid image (JPEG or PNG, at least 200x200)")
	default:
		logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
		platformerrors.Respond(c, http.StatusInternalServerError, "internal", "internal error")
	}
}

func (h *Handler) GetOwn(c *gin.Context) {
	p, err := h.service.GetOwn(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profiles.get_own", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, "profiles.upsert.bind", ErrInvalidInput)
		return
	}
	p, err := h.service.Upsert(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profiles.upsert", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, "profiles.prefs.bind", ErrInvalidInput)
		return
	}
	prefs, err := h.service.UpdatePreferences(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profiles.prefs", err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (h *Handler) ListInterests(c *gin.Context) {
	interests, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": interests})
}

func (h *Handler) GetPublic(c *gin.Context) {
	p, err := h.service.GetPublic(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		h.fail(c, "profiles.get_public", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) AddPhoto(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadBytes+(1<<20))
	file, err := c.FormFile("photo")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			h.fail(c, "profiles.photo.upload", ErrImageTooLarge)
			return
		}
		h.fail(c, "profiles.photo.upload", ErrInvalidImage)
		return
	}
	if file.Size > MaxUploadBytes {
		h.fail(c, "profiles.photo.upload", ErrImageTooLarge)
		return
	}
	var position *int
	if raw := c.PostForm("position"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			h.fail(c, "profiles.photo.upload", ErrInvalidInput)
			return
		}
		position = &n
	}
	f, err := file.Open()
	if err != nil {
		h.fail(c, "profiles.photo.upload", ErrInvalidImage)
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, MaxUploadBytes+1))
	if err != nil {
		h.fail(c, "profiles.photo.upload", ErrInvalidImage)
		return
	}
	photos, err := h.service.AddPhoto(c.Request.Context(), c.GetString("userID"), raw, position)
	if err != nil {
		h.fail(c, "profiles.photo.upload", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"photos": photos})
}

func (h *Handler) DeletePhoto(c *gin.Context) {
	photos, err := h.service.DeletePhoto(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		h.fail(c, "profiles.photo.delete", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

func (h *Handler) ReorderPhotos(c *gin.Context) {
	var req ReorderPhotosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, "profiles.photo.reorder.bind", ErrInvalidInput)
		return
	}
	photos, err := h.service.ReorderPhotos(c.Request.Context(), c.GetString("userID"), req.PhotoIDs)
	if err != nil {
		h.fail(c, "profiles.photo.reorder", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photos": photos})
}

// ServePhoto streams a photo. Photos are never publicly reachable: the request
// must be authenticated and pass the block check.
func (h *Handler) ServePhoto(c *gin.Context) {
	f, err := h.service.OpenPhoto(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	if err != nil {
		h.fail(c, "profiles.photo.serve", err)
		return
	}
	defer f.Close()
	c.Header("Content-Type", "image/jpeg")
	// A photo id never changes content (replacing creates a new id), so it can
	// be cached aggressively by the client.
	c.Header("Cache-Control", "private, max-age=86400, immutable")
	http.ServeContent(c.Writer, c.Request, "", time.Time{}, f)
}
