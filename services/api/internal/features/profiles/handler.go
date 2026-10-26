package profiles

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) respond(c *gin.Context, op string, status int, p OwnProfile, err error) {
	if err == nil {
		c.Header("Cache-Control", "no-store")
		c.JSON(status, p)
		return
	}
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		c.JSON(http.StatusBadRequest, gin.H{"error": verr.Message, "field": verr.Field})
		status = http.StatusBadRequest
	case errors.Is(err, ErrUnderage):
		c.JSON(http.StatusForbidden, gin.H{"error": ErrUnderage.Error(), "field": "birthDate"})
		status = http.StatusForbidden
	case errors.Is(err, ErrBirthDateChange):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrBirthDateChange.Error(), "field": "birthDate"})
		status = http.StatusBadRequest
	case errors.Is(err, ErrInvalidInterest):
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidInterest.Error(), "field": "interests"})
		status = http.StatusBadRequest
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		status = http.StatusNotFound
	case errors.Is(err, ErrRequiresProfile):
		c.JSON(http.StatusConflict, gin.H{"error": ErrRequiresProfile.Error()})
		status = http.StatusConflict
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		status = http.StatusInternalServerError
	}
	logger.LogHandlerError(c, op, status, err)
}

func (h *Handler) GetOwn(c *gin.Context) {
	p, err := h.service.GetOwn(c.Request.Context(), c.GetString("userID"))
	h.respond(c, "profiles.get", http.StatusOK, p, err)
}

func (h *Handler) Save(c *gin.Context) {
	var req SaveProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	p, err := h.service.Save(c.Request.Context(), c.GetString("userID"), req)
	h.respond(c, "profiles.save", http.StatusOK, p, err)
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var req SavePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	p, err := h.service.SavePreferences(c.Request.Context(), c.GetString("userID"), req)
	h.respond(c, "profiles.preferences", http.StatusOK, p, err)
}

func (h *Handler) SaveLocation(c *gin.Context) {
	var req SaveLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	p, err := h.service.SaveLocation(c.Request.Context(), c.GetString("userID"), req)
	h.respond(c, "profiles.location", http.StatusOK, p, err)
}

func (h *Handler) SavePrivacy(c *gin.Context) {
	var req SavePrivacyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	p, err := h.service.SavePrivacy(c.Request.Context(), c.GetString("userID"), req)
	h.respond(c, "profiles.privacy", http.StatusOK, p, err)
}

func (h *Handler) ListInterests(c *gin.Context) {
	items, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		logger.LogHandlerError(c, "profiles.interests", http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": items})
}
