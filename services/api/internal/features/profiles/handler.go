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

var validationErrors = []error{
	ErrInvalidFirstName, ErrInvalidBirthdate, ErrUnderage, ErrInvalidAge, ErrBirthdateLocked, ErrInvalidGender,
	ErrInvalidGoal, ErrBioTooLong, ErrOccupationTooLong, ErrCityTooLong, ErrInvalidInterestedIn, ErrInvalidAgeRange,
	ErrInvalidDistance, ErrTooManyInterests, ErrUnknownInterest, ErrInvalidCoordinates,
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	for _, known := range validationErrors {
		if errors.Is(err, known) {
			httpx.Fail(c, http.StatusBadRequest, known.Error())
			return
		}
	}
	switch {
	case errors.Is(err, ErrProfileNotFound):
		httpx.Fail(c, http.StatusNotFound, "profile not found")
	case errors.Is(err, ErrProfileNotAccessible):
		httpx.Fail(c, http.StatusNotFound, "profile not found")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Fail(c, http.StatusInternalServerError, "could not process profile")
	}
}

func (h *Handler) respond(c *gin.Context, op string, p OwnProfile, err error) {
	if err != nil {
		h.fail(c, op, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) GetOwn(c *gin.Context) {
	p, err := h.service.GetOwn(c.Request.Context(), httpx.UserID(c))
	h.respond(c, "profiles.get_own", p, err)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.Upsert(c.Request.Context(), httpx.UserID(c), req)
	h.respond(c, "profiles.upsert", p, err)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	h.respond(c, "profiles.preferences", p, err)
}

func (h *Handler) SetInterests(c *gin.Context) {
	var req InterestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.SetInterests(c.Request.Context(), httpx.UserID(c), req.InterestIDs)
	h.respond(c, "profiles.interests", p, err)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req LocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), *req.Latitude, *req.Longitude, req.City)
	h.respond(c, "profiles.location", p, err)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	p, err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c))
	h.respond(c, "profiles.location_clear", p, err)
}

func (h *Handler) UpdatePrivacy(c *gin.Context) {
	var req PrivacyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.UpdatePrivacy(c.Request.Context(), httpx.UserID(c), Privacy{
		Discoverable: *req.Discoverable, ShowAge: *req.ShowAge, ShowDistance: *req.ShowDistance,
	})
	h.respond(c, "profiles.privacy", p, err)
}

func (h *Handler) ListInterests(c *gin.Context) {
	interests, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests_list", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, InterestsResponse{Interests: interests})
}

func (h *Handler) GetPublic(c *gin.Context) {
	targetID := c.Param("userId")
	if !httpx.IsUUID(targetID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	p, err := h.service.GetPublic(c.Request.Context(), httpx.UserID(c), httpx.OrganizationID(c), targetID)
	if err != nil {
		h.fail(c, "profiles.get_public", err)
		return
	}
	c.JSON(http.StatusOK, p)
}
