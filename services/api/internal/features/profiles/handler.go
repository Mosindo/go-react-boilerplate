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

func (h *Handler) GetOwn(c *gin.Context) {
	profile, err := h.service.GetOwn(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "profiles.get_own", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "profiles.update.bind", err)
		return
	}
	profile, err := h.service.UpdateProfile(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.update", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "profiles.preferences.bind", err)
		return
	}
	profile, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.preferences", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) UpdateInterests(c *gin.Context) {
	var req UpdateInterestsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "profiles.interests.bind", err)
		return
	}
	profile, err := h.service.UpdateInterests(c.Request.Context(), httpx.UserID(c), req.InterestIDs)
	if err != nil {
		httpx.Fail(c, "profiles.interests", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "profiles.location.bind", err)
		return
	}
	profile, err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), *req.Latitude, *req.Longitude, req.City)
	if err != nil {
		httpx.Fail(c, "profiles.location", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	profile, err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "profiles.location.clear", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) ListInterests(c *gin.Context) {
	interests, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, "profiles.interests.list", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
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
