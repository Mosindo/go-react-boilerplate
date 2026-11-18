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

var validationErrors = []error{
	ErrInvalidFirstName, ErrInvalidBirthdate, ErrUnderage, ErrInvalidAge, ErrBirthdateLocked, ErrInvalidGender,
	ErrInvalidGoal, ErrInvalidHeight, ErrTextTooLong, ErrInvalidText, ErrInvalidPreferences, ErrInvalidLocation,
	ErrTooManyInterests, ErrUnknownInterest,
}

var errorCodes = map[error]string{
	ErrUnderage:        "underage",
	ErrBirthdateLocked: "birthdate_locked",
}

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	if errors.Is(err, ErrProfileRequired) {
		httpx.Fail(c, http.StatusConflict, "profile_required", err.Error(), operation, err)
		return
	}
	for _, known := range validationErrors {
		if errors.Is(err, known) {
			code := errorCodes[known]
			if code == "" {
				code = "validation_failed"
			}
			httpx.Fail(c, http.StatusBadRequest, code, err.Error(), operation, err)
			return
		}
	}
	httpx.Fail(c, http.StatusInternalServerError, "internal", "could not process profile", operation, err)
}

func (h *Handler) respondMe(c *gin.Context, operation string) {
	resp, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, operation, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetMe(c *gin.Context) {
	h.respondMe(c, "profiles.me")
}

func (h *Handler) UpsertMe(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "profiles.upsert.bind", err)
		return
	}
	if err := h.service.UpsertProfile(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "profiles.upsert", err)
		return
	}
	h.respondMe(c, "profiles.upsert")
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "profiles.preferences.bind", err)
		return
	}
	if _, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	h.respondMe(c, "profiles.preferences")
}

func (h *Handler) UpdatePrivacy(c *gin.Context) {
	var req PrivacyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "profiles.privacy.bind", err)
		return
	}
	if err := h.service.UpdatePrivacy(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "profiles.privacy", err)
		return
	}
	h.respondMe(c, "profiles.privacy")
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req LocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "profiles.location.bind", err)
		return
	}
	if err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "profiles.location", err)
		return
	}
	h.respondMe(c, "profiles.location")
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)); err != nil {
		h.fail(c, "profiles.location.clear", err)
		return
	}
	h.respondMe(c, "profiles.location.clear")
}

func (h *Handler) ListInterests(c *gin.Context) {
	interests, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests.catalog", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, InterestsResponse{Interests: interests})
}

func (h *Handler) ReplaceInterests(c *gin.Context) {
	var req InterestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "profiles.interests.bind", err)
		return
	}
	if _, err := h.service.ReplaceInterests(c.Request.Context(), httpx.UserID(c), req.Slugs); err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	h.respondMe(c, "profiles.interests")
}
