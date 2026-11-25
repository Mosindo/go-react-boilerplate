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

func (h *Handler) Me(c *gin.Context) {
	resp, err := h.service.Me(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "profiles.me", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateProfile(c *gin.Context) {
	var req UpdateProfileRequest
	if !httpx.BindJSON(c, "profiles.update", &req) {
		return
	}
	resp, err := h.service.UpdateProfile(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.update", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if !httpx.BindJSON(c, "profiles.preferences", &req) {
		return
	}
	resp, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.preferences", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdatePrivacy(c *gin.Context) {
	var req UpdatePrivacyRequest
	if !httpx.BindJSON(c, "profiles.privacy", &req) {
		return
	}
	resp, err := h.service.UpdatePrivacy(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.privacy", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if !httpx.BindJSON(c, "profiles.location", &req) {
		return
	}
	resp, err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "profiles.location", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, InterestsResponse{Interests: list})
}

func (h *Handler) Public(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	profile, err := h.service.Public(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		httpx.Fail(c, "profiles.public", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, profile)
}
