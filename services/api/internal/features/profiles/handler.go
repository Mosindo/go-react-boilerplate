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

func (h *Handler) GetMine(c *gin.Context) {
	profile, err := h.service.GetMine(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "profiles.get_mine", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) UpdateMine(c *gin.Context) {
	var req UpdateProfileRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	profile, err := h.service.UpdateMine(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.update_mine", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	profile, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.update_preferences", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), *req.Latitude, *req.Longitude); err != nil {
		httpx.Fail(c, "profiles.update_location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "profiles.clear_location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdatePrivacy(c *gin.Context) {
	var req UpdatePrivacyRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	profile, err := h.service.UpdatePrivacy(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.update_privacy", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) ListInterests(c *gin.Context) {
	interests, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, "profiles.list_interests", err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": interests})
}

func (h *Handler) GetPublic(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	profile, err := h.service.GetPublic(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		httpx.Fail(c, "profiles.get_public", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}
