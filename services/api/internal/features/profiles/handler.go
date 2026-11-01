package profiles

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(c, http.StatusBadRequest, "invalid input")
	case errors.Is(err, ErrUnderage):
		httpx.Error(c, http.StatusForbidden, "you must be at least 18 years old")
	case errors.Is(err, ErrProfileNotFound):
		httpx.Error(c, http.StatusNotFound, "profile not found")
	case errors.Is(err, ErrProfileRequired):
		httpx.Error(c, http.StatusConflict, "create your profile first")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) GetMyProfile(c *gin.Context) {
	p, err := h.service.GetProfile(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profiles.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) PutMyProfile(c *gin.Context) {
	var in ProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.SaveProfile(c.Request.Context(), c.GetString("userID"), in)
	if err != nil {
		h.fail(c, "profiles.save", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) GetSettings(c *gin.Context) {
	s, err := h.service.GetSettings(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profiles.settings.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, s)
}

func (h *Handler) PutSettings(c *gin.Context) {
	var in Settings
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	s, err := h.service.SaveSettings(c.Request.Context(), c.GetString("userID"), in)
	if err != nil {
		h.fail(c, "profiles.settings.save", err)
		return
	}
	c.JSON(http.StatusOK, s)
}

func (h *Handler) ListInterests(c *gin.Context) {
	items, err := h.service.Interests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) GetPublicProfile(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	p, err := h.service.PublicProfile(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		h.fail(c, "profiles.public", err)
		return
	}
	c.JSON(http.StatusOK, p)
}
