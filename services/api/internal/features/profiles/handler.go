package profiles

import (
	"errors"
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func writeError(c *gin.Context, op string, err error) {
	var validation *platformerrors.ValidationError
	status := http.StatusInternalServerError
	body := gin.H{"error": "internal error"}
	switch {
	case errors.As(err, &validation):
		status = http.StatusBadRequest
		body = gin.H{"error": validation.Message, "field": validation.Field}
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
		body = gin.H{"error": "profile not found"}
	}
	logger.LogHandlerError(c, op, status, err)
	c.JSON(status, body)
}

func (h *Handler) GetMine(c *gin.Context) {
	profile, err := h.service.GetSelf(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		writeError(c, "profile.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) UpdateMine(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	profile, err := h.service.UpdateSelf(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		writeError(c, "profile.update", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) GetPreferences(c *gin.Context) {
	prefs, err := h.service.GetPreferences(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		writeError(c, "preferences.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, prefs)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	prefs, err := h.service.UpdatePreferences(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		writeError(c, "preferences.update", err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (h *Handler) ListInterests(c *gin.Context) {
	interests, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		writeError(c, "interests.list", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": interests})
}

func (h *Handler) GetPublic(c *gin.Context) {
	targetID := c.Param("userId")
	if !IsUUID(targetID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}
	profile, err := h.service.GetPublic(c.Request.Context(), c.GetString("userID"), targetID)
	if err != nil {
		writeError(c, "profile.public", err)
		return
	}
	c.Header("Cache-Control", "private, no-cache")
	c.JSON(http.StatusOK, profile)
}
