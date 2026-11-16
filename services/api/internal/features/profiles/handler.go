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

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status, code, msg := http.StatusInternalServerError, "internal", "something went wrong"
	var v ValidationError
	switch {
	case errors.As(err, &v):
		status, code, msg = http.StatusBadRequest, "validation_failed", v.Message
	case errors.Is(err, ErrNotFound):
		status, code, msg = http.StatusConflict, "profile_required", "create your profile first"
	case errors.Is(err, ErrForbidden):
		status, code, msg = http.StatusNotFound, "not_found", "profile not found"
	}
	logger.LogHandlerError(c, op, status, err)
	httpx.Fail(c, status, code, msg)
}

func (h *Handler) Interests(c *gin.Context) {
	items, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": items})
}

func (h *Handler) Me(c *gin.Context) {
	resp, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "profiles.me", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SaveProfile(c *gin.Context) {
	var in ProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	resp, err := h.service.SaveProfile(c.Request.Context(), httpx.UserID(c), in)
	if err != nil {
		h.fail(c, "profiles.save", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var in PreferencesInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request")
		return
	}
	resp, err := h.service.SavePreferences(c.Request.Context(), httpx.UserID(c), in)
	if err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var in LocationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "latitude and longitude are required")
		return
	}
	if err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), *in.Latitude, *in.Longitude); err != nil {
		h.fail(c, "profiles.location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) SetVisibility(c *gin.Context) {
	var in VisibilityInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "isDiscoverable is required")
		return
	}
	if err := h.service.SetDiscoverable(c.Request.Context(), httpx.UserID(c), *in.IsDiscoverable); err != nil {
		h.fail(c, "profiles.visibility", err)
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
	if err != nil {
		h.fail(c, "profiles.view", err)
		return
	}
	c.JSON(http.StatusOK, card)
}
