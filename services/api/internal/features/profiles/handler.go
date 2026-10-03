package profiles

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/validate"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (h *Handler) GetOwn(c *gin.Context) {
	p, err := h.service.GetOwn(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profiles.get_own", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	p, err := h.service.Upsert(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profiles.upsert", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) SetLocation(c *gin.Context) {
	var req LocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.service.SetLocation(c.Request.Context(), c.GetString("userID"), req.Latitude, req.Longitude); err != nil {
		h.fail(c, "profiles.set_location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) SetPreferences(c *gin.Context) {
	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	p, err := h.service.SetPreferences(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profiles.set_preferences", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) SetPrivacy(c *gin.Context) {
	var req PrivacyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.service.SetPrivacy(c.Request.Context(), c.GetString("userID"), *req.IsVisible, *req.ShowDistance); err != nil {
		h.fail(c, "profiles.set_privacy", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetPublic(c *gin.Context) {
	if !validate.IsUUID(c.Param("userId")) {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}
	p, err := h.service.GetPublic(c.Request.Context(), c.GetString("userID"), c.Param("userId"))
	if err != nil {
		h.fail(c, "profiles.get_public", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) ListInterests(c *gin.Context) {
	items, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.list_interests", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"interests": items})
}
