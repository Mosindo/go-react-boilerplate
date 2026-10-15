package users

import (
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

func (h *Handler) Me(c *gin.Context) {
	resp, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "users.me", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SaveProfile(c *gin.Context) {
	var in ProfileInput
	if !httpx.Bind(c, &in) {
		return
	}
	if err := h.service.SaveProfile(c.Request.Context(), httpx.UserID(c), in); err != nil {
		httpx.Fail(c, "users.save_profile", err)
		return
	}
	h.Me(c)
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var in PreferencesInput
	if !httpx.Bind(c, &in) {
		return
	}
	if err := h.service.SavePreferences(c.Request.Context(), httpx.UserID(c), in); err != nil {
		httpx.Fail(c, "users.save_preferences", err)
		return
	}
	h.Me(c)
}

func (h *Handler) SetLocation(c *gin.Context) {
	var in LocationInput
	if !httpx.Bind(c, &in) {
		return
	}
	if err := h.service.SetLocation(c.Request.Context(), httpx.UserID(c), in); err != nil {
		httpx.Fail(c, "users.set_location", err)
		return
	}
	h.Me(c)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "users.clear_location", err)
		return
	}
	h.Me(c)
}

func (h *Handler) SetPrivacy(c *gin.Context) {
	var in PrivacyInput
	if !httpx.Bind(c, &in) {
		return
	}
	if err := h.service.SetPrivacy(c.Request.Context(), httpx.UserID(c), in); err != nil {
		httpx.Fail(c, "users.set_privacy", err)
		return
	}
	h.Me(c)
}

func (h *Handler) Interests(c *gin.Context) {
	interests, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, "users.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": interests})
}
