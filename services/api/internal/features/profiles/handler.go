package profiles

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) Me(c *gin.Context) {
	p, err := h.service.Own(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "profiles.me", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	p, err := h.service.UpsertProfile(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.upsert", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Preferences(c *gin.Context) {
	var req PreferencesRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	p, err := h.service.SetPreferences(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.preferences", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Location(c *gin.Context) {
	var req LocationRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	p, err := h.service.SetLocation(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.location", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": list})
}

func (h *Handler) Public(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	p, err := h.service.Public(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "profiles.public", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}
