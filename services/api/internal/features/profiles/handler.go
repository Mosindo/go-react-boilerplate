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

var validationErrors = map[error]string{
	ErrInvalidFirstName:    "invalid_first_name",
	ErrInvalidBirthdate:    "invalid_birthdate",
	ErrUnderage:            "underage",
	ErrInvalidAge:          "invalid_birthdate",
	ErrInvalidGender:       "invalid_gender",
	ErrBioTooLong:          "bio_too_long",
	ErrCityTooLong:         "city_too_long",
	ErrTooManyInterests:    "too_many_interests",
	ErrUnknownInterest:     "unknown_interest",
	ErrInvalidInterestedIn: "invalid_interested_in",
	ErrInvalidAgeRange:     "invalid_age_range",
	ErrInvalidDistance:     "invalid_distance",
	ErrInvalidCoordinates:  "invalid_coordinates",
}

func (h *Handler) respond(c *gin.Context, operation string, profile MyProfile, err error) {
	if err != nil {
		for known, code := range validationErrors {
			if errors.Is(err, known) {
				httpx.BadRequest(c, code, known.Error())
				return
			}
		}
		if errors.Is(err, ErrProfileNotFound) {
			httpx.Fail(c, http.StatusNotFound, "not_found", "profile not found", operation, err)
			return
		}
		httpx.Internal(c, operation, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) GetMine(c *gin.Context) {
	profile, err := h.service.GetMine(c.Request.Context(), httpx.UserID(c))
	h.respond(c, "profiles.get_mine", profile, err)
}

func (h *Handler) UpdateMine(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "invalid profile payload")
		return
	}
	profile, err := h.service.UpdateMine(c.Request.Context(), httpx.UserID(c), req)
	h.respond(c, "profiles.update", profile, err)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "invalid preferences payload")
		return
	}
	profile, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	h.respond(c, "profiles.preferences", profile, err)
}

func (h *Handler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "invalid settings payload")
		return
	}
	profile, err := h.service.UpdateSettings(c.Request.Context(), httpx.UserID(c), Settings{
		IsDiscoverable: *req.IsDiscoverable,
		HideDistance:   *req.HideDistance,
	})
	h.respond(c, "profiles.settings", profile, err)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "latitude and longitude are required")
		return
	}
	profile, err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), *req.Latitude, *req.Longitude, req.City)
	h.respond(c, "profiles.location", profile, err)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	profile, err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c))
	h.respond(c, "profiles.location_clear", profile, err)
}

func (h *Handler) ListInterests(c *gin.Context) {
	interests, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		httpx.Internal(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": interests})
}

func (h *Handler) GetPublic(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	profile, err := h.service.GetPublic(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		if errors.Is(err, ErrNotVisible) {
			httpx.Fail(c, http.StatusNotFound, "not_found", "profile not available", "profiles.public", nil)
			return
		}
		httpx.Internal(c, "profiles.public", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, profile)
}
