package profiles

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListInterests(c *gin.Context) {
	interests, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		httpx.Respond(c, "profiles.interests", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"interests": interests})
}

func (h *Handler) GetMine(c *gin.Context) {
	res, err := h.service.GetMine(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Respond(c, "profiles.get_mine", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, res)
}

func (h *Handler) UpsertMine(c *gin.Context) {
	var in ProfileInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	profile, err := h.service.UpsertMine(c.Request.Context(), httpx.UserID(c), in)
	if err != nil {
		httpx.Respond(c, "profiles.upsert", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var in Preferences
	if !httpx.BindJSON(c, &in) {
		return
	}
	prefs, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), in)
	if err != nil {
		httpx.Respond(c, "profiles.preferences", err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var in LocationInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	if err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), *in.Latitude, *in.Longitude); err != nil {
		httpx.Respond(c, "profiles.location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetProfile(c *gin.Context) {
	profile, err := h.service.GetVisible(c.Request.Context(), httpx.UserID(c), c.Param("userId"))
	if err != nil {
		httpx.Respond(c, "profiles.get", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}
