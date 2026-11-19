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

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Interests(c *gin.Context) {
	items, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Internal(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": items})
}

func (h *Handler) GetOwn(c *gin.Context) {
	p, err := h.service.Own(c.Request.Context(), httpx.UserID(c))
	if errors.Is(err, ErrNotFound) {
		httpx.Fail(c, http.StatusNotFound, "profile_not_found", "profile not created yet")
		return
	}
	if err != nil {
		httpx.Internal(c, "profiles.get_own", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) UpsertOwn(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "firstName, birthdate and gender are required")
		return
	}
	p, err := h.service.Upsert(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "profiles.upsert", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "interestedIn, ageMin, ageMax, maxDistanceKm, isDiscoverable and showDistance are required")
		return
	}
	prefs, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "latitude and longitude are required")
		return
	}
	if err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "profiles.location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Internal(c, "profiles.location_clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) View(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	card, err := h.service.View(c.Request.Context(), httpx.UserID(c), targetID)
	if errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotFound) {
		// Same answer for "does not exist" and "not allowed" to avoid enumeration.
		httpx.Fail(c, http.StatusNotFound, "profile_not_found", "profile not available")
		return
	}
	if err != nil {
		httpx.Internal(c, "profiles.view", err)
		return
	}
	c.JSON(http.StatusOK, card)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusConflict, "profile_required", "create your profile first")
	case errors.Is(err, ErrUnderage):
		httpx.Fail(c, http.StatusUnprocessableEntity, "underage", err.Error())
	case errors.Is(err, ErrBirthdateLocked):
		httpx.Fail(c, http.StatusUnprocessableEntity, "birthdate_locked", err.Error())
	case errors.Is(err, ErrInvalidFirstName), errors.Is(err, ErrInvalidBirthdate), errors.Is(err, ErrInvalidAge),
		errors.Is(err, ErrInvalidGender), errors.Is(err, ErrInvalidBio), errors.Is(err, ErrInvalidJobTitle),
		errors.Is(err, ErrInvalidCity), errors.Is(err, ErrInvalidLookingFor), errors.Is(err, ErrTooManyInterests),
		errors.Is(err, ErrInvalidInterest), errors.Is(err, ErrInvalidPrefs), errors.Is(err, ErrInvalidLocation):
		httpx.BadRequest(c, err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}
