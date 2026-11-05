package profiles

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

// fail maps service errors to HTTP responses.
func fail(c *gin.Context, op string, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.BadRequest(c, ve.Message)
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusConflict, "create your profile first")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) Interests(c *gin.Context) {
	items, err := h.service.Interests(c.Request.Context())
	if err != nil {
		fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": items})
}

func (h *Handler) Overview(c *gin.Context) {
	o, err := h.service.Overview(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		fail(c, "profiles.overview", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, o)
}

// saveAndReturn runs a mutation then answers with the fresh overview.
func (h *Handler) saveAndReturn(c *gin.Context, op string, save func() error) {
	if err := save(); err != nil {
		fail(c, op, err)
		return
	}
	h.Overview(c)
}

func (h *Handler) SaveProfile(c *gin.Context) {
	var in ProfileInput
	if !httpx.Bind(c, &in) {
		return
	}
	h.saveAndReturn(c, "profiles.save", func() error { return h.service.SaveProfile(c.Request.Context(), httpx.UserID(c), in) })
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var in Preferences
	if !httpx.Bind(c, &in) {
		return
	}
	h.saveAndReturn(c, "profiles.preferences", func() error { return h.service.SavePreferences(c.Request.Context(), httpx.UserID(c), in) })
}

func (h *Handler) SavePrivacy(c *gin.Context) {
	var in Privacy
	if !httpx.Bind(c, &in) {
		return
	}
	h.saveAndReturn(c, "profiles.privacy", func() error { return h.service.SavePrivacy(c.Request.Context(), httpx.UserID(c), in) })
}

func (h *Handler) SaveLocation(c *gin.Context) {
	var in LocationInput
	if !httpx.Bind(c, &in) {
		return
	}
	h.saveAndReturn(c, "profiles.location", func() error { return h.service.SaveLocation(c.Request.Context(), httpx.UserID(c), in) })
}

func (h *Handler) ClearLocation(c *gin.Context) {
	h.saveAndReturn(c, "profiles.location.clear", func() error { return h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)) })
}
