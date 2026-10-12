package profiles

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.Fail(c, http.StatusUnprocessableEntity, ve.Message)
	case errors.Is(err, ErrBirthDateLocked):
		httpx.Fail(c, http.StatusConflict, "birth date cannot be changed")
	case errors.Is(err, ErrProfileMissing):
		httpx.Fail(c, http.StatusConflict, "create your profile first")
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "profile not found")
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Interests(c *gin.Context) {
	items, err := h.service.Interests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Me(c *gin.Context) {
	res, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profiles.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpdateProfile(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "profiles.update", err)
		return
	}
	h.Me(c)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpdatePreferences(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	h.Me(c)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpdateLocation(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "profiles.location", err)
		return
	}
	h.Me(c)
}

func (h *Handler) UpdatePrivacy(c *gin.Context) {
	var req UpdatePrivacyRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpdatePrivacy(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "profiles.privacy", err)
		return
	}
	h.Me(c)
}

func (h *Handler) Get(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	card, err := h.service.ViewableCard(c.Request.Context(), c.GetString("userID"), targetID)
	if err != nil {
		h.fail(c, "profiles.get", err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, card)
}
