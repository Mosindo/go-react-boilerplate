package profiles

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Interests(c *gin.Context) {
	items, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, "profiles.interests", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"interests": items})
}

func (h *Handler) GetMine(c *gin.Context) {
	me, err := h.service.Mine(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "profiles.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, me)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	me, err := h.service.Upsert(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.upsert", err)
		return
	}
	c.JSON(http.StatusOK, me)
}

func (h *Handler) SetLocation(c *gin.Context) {
	var req LocationRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.SetLocation(c.Request.Context(), httpx.UserID(c), req); err != nil {
		httpx.Fail(c, "profiles.location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "profiles.location.clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) SetPrivacy(c *gin.Context) {
	var req PrivacyRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.SetPrivacy(c.Request.Context(), httpx.UserID(c), req); err != nil {
		httpx.Fail(c, "profiles.privacy", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) SetPreferences(c *gin.Context) {
	var req PreferencesRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.SetPreferences(c.Request.Context(), httpx.UserID(c), req); err != nil {
		httpx.Fail(c, "profiles.preferences", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetCard(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	card, err := h.service.Card(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "profiles.card", err)
		return
	}
	c.JSON(http.StatusOK, card)
}
