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

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case IsValidation(err):
		httpx.Fail(c, op, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, ErrBirthDateLocked):
		httpx.Fail(c, op, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, ErrProfileRequired):
		httpx.Fail(c, op, http.StatusConflict, err.Error(), err)
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "profile not found", err)
	default:
		httpx.Fail(c, op, http.StatusInternalServerError, "internal error", err)
	}
}

func (h *Handler) Me(c *gin.Context) {
	resp, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profiles.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SaveProfile(c *gin.Context) {
	var in ProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, "profiles.save.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := h.service.SaveProfile(c.Request.Context(), c.GetString("userID"), in); err != nil {
		h.fail(c, "profiles.save", err)
		return
	}
	h.Me(c)
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var in PreferencesInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, "profiles.preferences.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := h.service.SavePreferences(c.Request.Context(), c.GetString("userID"), in); err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	h.Me(c)
}

func (h *Handler) SavePrivacy(c *gin.Context) {
	var in PrivacyInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, "profiles.privacy.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := h.service.SavePrivacy(c.Request.Context(), c.GetString("userID"), in); err != nil {
		h.fail(c, "profiles.privacy", err)
		return
	}
	h.Me(c)
}

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.Interests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": list})
}

func (h *Handler) View(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		httpx.Fail(c, "profiles.view.id", http.StatusBadRequest, "invalid user id", errors.New("invalid user id"))
		return
	}
	profile, err := h.service.View(c.Request.Context(), c.GetString("userID"), targetID)
	if err != nil {
		h.fail(c, "profiles.view", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, profile)
}
