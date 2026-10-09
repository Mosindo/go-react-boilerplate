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
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.BadRequest(c, ve.Message)
	case errors.Is(err, ErrNotFound):
		httpx.NotFound(c)
	case errors.Is(err, ErrViewerIncomplete):
		httpx.Fail(c, http.StatusForbidden, httpx.CodeProfileIncomplete, "complete your profile first")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) Me(c *gin.Context) {
	me, err := h.service.Me(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Fail(c, http.StatusUnauthorized, httpx.CodeUnauthorized, "user not found")
			return
		}
		h.fail(c, "profiles.me", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, me)
}

func (h *Handler) GetProfile(c *gin.Context) {
	p, err := h.service.GetProfile(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profiles.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	p, err := h.service.UpdateProfile(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profiles.update", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) GetPreferences(c *gin.Context) {
	p, err := h.service.GetPreferences(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profiles.prefs.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req Preferences
	if !httpx.BindJSON(c, &req) {
		return
	}
	p, err := h.service.UpdatePreferences(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profiles.prefs.update", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) ListInterests(c *gin.Context) {
	list, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": list})
}

func (h *Handler) PublicProfile(c *gin.Context) {
	p, err := h.service.PublicProfile(c.Request.Context(), c.GetString("userID"), c.Param("userId"))
	if err != nil {
		h.fail(c, "profiles.public", err)
		return
	}
	c.Header("Cache-Control", "private, no-cache")
	c.JSON(http.StatusOK, p)
}
