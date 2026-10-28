package profiles

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": list})
}

func (h *Handler) Get(c *gin.Context) {
	p, err := h.service.Own(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "profiles.get", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Save(c *gin.Context) {
	var in ProfileInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	p, err := h.service.Save(c.Request.Context(), httpx.UserID(c), in)
	if err != nil {
		httpx.Fail(c, "profiles.save", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) SaveLocation(c *gin.Context) {
	var in LocationInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	p, err := h.service.SaveLocation(c.Request.Context(), httpx.UserID(c), in)
	if err != nil {
		httpx.Fail(c, "profiles.location", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) GetPreferences(c *gin.Context) {
	p, err := h.service.Preferences(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "profiles.preferences.get", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var in Preferences
	if !httpx.BindJSON(c, &in) {
		return
	}
	p, err := h.service.SavePreferences(c.Request.Context(), httpx.UserID(c), in)
	if err != nil {
		httpx.Fail(c, "profiles.preferences.save", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Public(c *gin.Context) {
	target, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	p, err := h.service.Public(c.Request.Context(), httpx.UserID(c), target)
	if err != nil {
		httpx.Fail(c, "profiles.public", err)
		return
	}
	c.JSON(http.StatusOK, p)
}
