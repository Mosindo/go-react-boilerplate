package profiles

import (
	"errors"
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

func (h *Handler) Me(c *gin.Context) {
	resp, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "profiles.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpsertProfile(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.InvalidRequest(c)
		return
	}
	if err := h.service.UpsertProfile(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "profiles.upsert", err)
		return
	}
	h.Me(c)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.InvalidRequest(c)
		return
	}
	if err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	h.Me(c)
}

func (h *Handler) UpdateInterests(c *gin.Context) {
	var req UpdateInterestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.InvalidRequest(c)
		return
	}
	if err := h.service.UpdateInterests(c.Request.Context(), httpx.UserID(c), req.InterestIDs); err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	h.Me(c)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.InvalidRequest(c)
		return
	}
	if err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), *req.Latitude, *req.Longitude); err != nil {
		h.fail(c, "profiles.location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)); err != nil {
		h.fail(c, "profiles.location.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.InvalidRequest(c)
		return
	}
	if err := h.service.UpdateSettings(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "profiles.settings", err)
		return
	}
	h.Me(c)
}

func (h *Handler) Interests(c *gin.Context) {
	interests, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests.list", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, InterestsResponse{Interests: interests})
}

func (h *Handler) View(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		httpx.Error(c, http.StatusBadRequest, "invalid_user_id", "invalid user id")
		return
	}
	card, err := h.service.View(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		h.fail(c, "profiles.view", err)
		return
	}
	c.JSON(http.StatusOK, card)
}

var validationErrors = []error{
	ErrInvalidFirstName, ErrInvalidBirthdate, ErrInvalidAge, ErrInvalidGender, ErrBioTooLong, ErrCityTooLong,
	ErrJobTooLong, ErrInvalidHeight, ErrInvalidGoal, ErrInvalidInterestedIn, ErrInvalidAgeRange, ErrInvalidDistance,
	ErrTooManyInterests, ErrUnknownInterest, ErrInvalidLocation,
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, ErrTooYoung):
		status = http.StatusUnprocessableEntity
		httpx.Error(c, status, "underage", err.Error())
	case errors.Is(err, ErrProfileNotFound):
		status = http.StatusNotFound
		httpx.Error(c, status, "profile_not_found", "profile not found")
	case errors.Is(err, ErrUserNotFound):
		status = http.StatusUnauthorized
		httpx.Error(c, status, "user_not_found", "user not found")
	default:
		for _, v := range validationErrors {
			if errors.Is(err, v) {
				status = http.StatusBadRequest
				httpx.Error(c, status, "validation_error", err.Error())
				logger.LogHandlerError(c, op, status, err)
				return
			}
		}
		httpx.Internal(c, "could not process profile request")
	}
	logger.LogHandlerError(c, op, status, err)
}
