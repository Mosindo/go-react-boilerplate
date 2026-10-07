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

func (h *Handler) Interests(c *gin.Context) {
	items, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Respond(c, "profiles.interests", err)
		return
	}
	c.JSON(http.StatusOK, httpx.Items(items))
}

func (h *Handler) GetMe(c *gin.Context) {
	profile, err := h.service.GetSelf(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Respond(c, "profiles.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) PutMe(c *gin.Context) {
	var req UpsertProfileRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	profile, err := h.service.Upsert(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Respond(c, "profiles.upsert", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) PutLocation(c *gin.Context) {
	var req LocationRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.SetLocation(c.Request.Context(), httpx.UserID(c), *req.Latitude, *req.Longitude); err != nil {
		httpx.Respond(c, "profiles.location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetPreferences(c *gin.Context) {
	prefs, err := h.service.GetPreferences(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Respond(c, "profiles.preferences.get", err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (h *Handler) PutPreferences(c *gin.Context) {
	var req Preferences
	if !httpx.BindJSON(c, &req) {
		return
	}
	prefs, err := h.service.SavePreferences(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Respond(c, "profiles.preferences.put", err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}
