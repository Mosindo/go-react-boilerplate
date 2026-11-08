package profiles

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) GetMe(c *gin.Context) {
	me, err := h.service.GetMe(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Fail(c, "profiles.get_me", err)
		return
	}
	c.JSON(http.StatusOK, me)
}

func (h *Handler) UpdateMe(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	me, err := h.service.UpdateProfile(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		httpx.Fail(c, "profiles.update", err)
		return
	}
	c.JSON(http.StatusOK, me)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	me, err := h.service.UpdateLocation(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		httpx.Fail(c, "profiles.location", err)
		return
	}
	c.JSON(http.StatusOK, me)
}

func (h *Handler) GetPublic(c *gin.Context) {
	card, err := h.service.GetPublic(c.Request.Context(), c.GetString("userID"), c.Param("userId"))
	if err != nil {
		httpx.Fail(c, "profiles.get_public", err)
		return
	}
	c.JSON(http.StatusOK, card)
}

func (h *Handler) ListInterests(c *gin.Context) {
	items, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, "profiles.interests", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"interests": items})
}

func (h *Handler) GetPreferences(c *gin.Context) {
	p, err := h.service.GetPreferences(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Fail(c, "profiles.get_preferences", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	p, err := h.service.UpdatePreferences(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		httpx.Fail(c, "profiles.update_preferences", err)
		return
	}
	c.JSON(http.StatusOK, p)
}
