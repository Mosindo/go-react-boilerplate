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
		httpx.Fail(c, "profiles.get_own", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Save(c *gin.Context) {
	var in ProfileInput
	if !httpx.Bind(c, &in) {
		return
	}
	p, err := h.service.Save(c.Request.Context(), httpx.UserID(c), in)
	if err != nil {
		httpx.Fail(c, "profiles.save", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"items": list})
}

func (h *Handler) GetPreferences(c *gin.Context) {
	p, err := h.service.GetPreferences(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "profiles.get_preferences", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var in Preferences
	if !httpx.Bind(c, &in) {
		return
	}
	p, err := h.service.SavePreferences(c.Request.Context(), httpx.UserID(c), in)
	if err != nil {
		httpx.Fail(c, "profiles.save_preferences", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) View(c *gin.Context) {
	p, err := h.service.View(c.Request.Context(), httpx.UserID(c), c.Param("id"))
	if err != nil {
		httpx.Fail(c, "profiles.view", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}
