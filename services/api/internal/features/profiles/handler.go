package profiles

import (
	"errors"
	"net/http"

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
	case IsValidationError(err):
		httpx.FailCode(c, operation, http.StatusBadRequest, "validation", err.Error(), err)
	case errors.Is(err, ErrProfileNotFound), errors.Is(err, ErrProfileHidden):
		httpx.Fail(c, operation, http.StatusNotFound, "profile not found", err)
	default:
		httpx.Fail(c, operation, http.StatusInternalServerError, "could not process profile", err)
	}
}

func (h *Handler) GetOwn(c *gin.Context) {
	profile, err := h.service.GetOwn(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "profiles.get_own", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "profiles.update.bind", "invalid request", err)
		return
	}
	profile, err := h.service.Update(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "profiles.update", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "profiles.location.bind", "latitude and longitude are required", err)
		return
	}
	profile, err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), *req.Latitude, *req.Longitude, req.City)
	if err != nil {
		h.fail(c, "profiles.location", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "profiles.preferences.bind", "invalid request", err)
		return
	}
	profile, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) Interests(c *gin.Context) {
	interests, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": interests})
}

func (h *Handler) GetPublic(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId", "profiles.get_public.validate")
	if !ok {
		return
	}
	profile, err := h.service.GetPublic(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		h.fail(c, "profiles.get_public", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}
