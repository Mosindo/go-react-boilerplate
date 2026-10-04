package profiles

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) GetOwn(c *gin.Context) {
	p, err := h.service.GetOwn(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "profile.get", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Save(c *gin.Context) {
	var req SaveProfileRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	p, err := h.service.Save(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profile.save", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) SetLocation(c *gin.Context) {
	var req LocationRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.SetLocation(c.Request.Context(), httpx.UserID(c), req); err != nil {
		httpx.Fail(c, "profile.location", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "profile.location.clear", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) GetPreferences(c *gin.Context) {
	p, err := h.service.GetPreferences(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "preferences.get", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var req Preferences
	if !httpx.BindJSON(c, &req) {
		return
	}
	p, err := h.service.SavePreferences(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "preferences.save", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) ListInterests(c *gin.Context) {
	items, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, "interests.list", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": items})
}

func (h *Handler) GetVisible(c *gin.Context) {
	p, err := h.service.GetVisible(c.Request.Context(), httpx.UserID(c), c.Param("userId"))
	if err != nil {
		httpx.Fail(c, "profile.public", err)
		return
	}
	c.JSON(http.StatusOK, p)
}
