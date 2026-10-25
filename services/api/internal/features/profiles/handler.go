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

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	var ve ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.BadRequest(c, ve.Message)
	case errors.Is(err, ErrUnderage):
		httpx.Fail(c, http.StatusBadRequest, "underage", err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.NotFound(c)
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) Get(c *gin.Context) {
	state, err := h.service.Get(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profile.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, state)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	p, err := h.service.Upsert(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profile.upsert", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) SetPreferences(c *gin.Context) {
	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	prefs, err := h.service.SetPreferences(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "preferences.set", err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (h *Handler) SetLocation(c *gin.Context) {
	var req LocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	p, err := h.service.SetLocation(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profile.location", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), c.GetString("userID")); err != nil {
		h.fail(c, "profile.location.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Interests(c *gin.Context) {
	items, err := h.service.Interests(c.Request.Context())
	if err != nil {
		h.fail(c, "interests.list", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) View(c *gin.Context) {
	target, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	p, err := h.service.View(c.Request.Context(), c.GetString("userID"), target)
	if err != nil {
		h.fail(c, "profile.view", err)
		return
	}
	c.JSON(http.StatusOK, p)
}
