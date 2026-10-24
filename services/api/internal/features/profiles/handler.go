package profiles

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	var ve ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.Fail(c, op, http.StatusUnprocessableEntity, ve.Message, err)
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "profile not found", err)
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Interests(c *gin.Context) {
	items, err := h.service.Interests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": items})
}

func (h *Handler) Own(c *gin.Context) {
	own, err := h.service.Own(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profiles.own", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, own)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if !httpx.BindJSON(c, "profiles.upsert", &req) {
		return
	}
	own, err := h.service.Upsert(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profiles.upsert", err)
		return
	}
	c.JSON(http.StatusOK, own)
}

func (h *Handler) SetPreferences(c *gin.Context) {
	var req Preferences
	if !httpx.BindJSON(c, "profiles.preferences", &req) {
		return
	}
	prefs, err := h.service.SetPreferences(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (h *Handler) SetLocation(c *gin.Context) {
	var req LocationRequest
	if !httpx.BindJSON(c, "profiles.location", &req) {
		return
	}
	if err := h.service.SetLocation(c.Request.Context(), c.GetString("userID"), *req.Latitude, *req.Longitude); err != nil {
		h.fail(c, "profiles.location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), c.GetString("userID")); err != nil {
		h.fail(c, "profiles.location.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Public(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	card, err := h.service.Card(c.Request.Context(), c.GetString("userID"), targetID)
	if err != nil {
		h.fail(c, "profiles.public", err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, card)
}
