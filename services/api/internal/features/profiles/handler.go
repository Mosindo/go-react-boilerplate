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
	list, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"interests": list})
}

func (h *Handler) GetProfile(c *gin.Context) {
	p, err := h.service.GetProfile(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) PutProfile(c *gin.Context) {
	var in ProfileInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	p, err := h.service.SaveProfile(c.Request.Context(), httpx.UserID(c), in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) GetPreferences(c *gin.Context) {
	p, err := h.service.GetPreferences(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) PutPreferences(c *gin.Context) {
	var in Preferences
	if !httpx.BindJSON(c, &in) {
		return
	}
	p, err := h.service.SetPreferences(c.Request.Context(), httpx.UserID(c), in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}
