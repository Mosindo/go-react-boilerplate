package profiles

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetOwn(c *gin.Context) {
	p, err := h.service.GetOwn(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "profiles.get_own", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Update(c *gin.Context) {
	var patch ProfilePatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		httpx.BadRequest(c, "invalid_request", "invalid request")
		return
	}
	p, err := h.service.Update(c.Request.Context(), httpx.UserID(c), patch)
	if err != nil {
		h.fail(c, "profiles.update", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req LocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "invalid request")
		return
	}
	p, err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "profiles.update_location", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_preferences", "invalid preferences")
		return
	}
	p, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "profiles.update_preferences", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Internal(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, InterestsResponse{Interests: list})
}

func (h *Handler) Public(c *gin.Context) {
	targetID := strings.TrimSpace(c.Param("userId"))
	if !httpx.IsUUID(targetID) {
		httpx.BadRequest(c, "invalid_user_id", "invalid user id")
		return
	}
	card, err := h.service.PublicProfile(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		h.fail(c, "profiles.public", err)
		return
	}
	c.JSON(http.StatusOK, card)
}

var validationErrors = map[error]string{
	ErrInvalidFirstName:   "invalid_first_name",
	ErrInvalidBirthdate:   "invalid_birthdate",
	ErrUnderage:           "underage",
	ErrBirthdateLocked:    "birthdate_locked",
	ErrInvalidGender:      "invalid_gender",
	ErrBioTooLong:         "bio_too_long",
	ErrJobTitleTooLong:    "job_title_too_long",
	ErrTooManyInterests:   "too_many_interests",
	ErrUnknownInterest:    "unknown_interest",
	ErrInvalidLocation:    "invalid_location",
	ErrCityTooLong:        "city_too_long",
	ErrInvalidPreferences: "invalid_preferences",
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	for target, code := range validationErrors {
		if errors.Is(err, target) {
			httpx.Fail(c, op, http.StatusBadRequest, code, target.Error(), nil)
			return
		}
	}
	switch {
	case errors.Is(err, ErrProfileNotFound):
		httpx.Fail(c, op, http.StatusUnauthorized, "user_not_found", "user not found", err)
	case errors.Is(err, ErrNotVisible):
		httpx.Fail(c, op, http.StatusNotFound, "profile_not_found", "profile not found", nil)
	default:
		httpx.Internal(c, op, err)
	}
}
