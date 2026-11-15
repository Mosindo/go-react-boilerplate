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
	interests, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": interests})
}

func (h *Handler) GetOwn(c *gin.Context) {
	resp, err := h.service.GetOwn(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "profiles.get_own", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateOwn(c *gin.Context) {
	var req UpdateProfileRequest
	if !httpx.Bind(c, "profiles.update", &req) {
		return
	}
	h.respond(c, "profiles.update", func() (MeProfileResponse, error) {
		return h.service.UpdateOwn(c.Request.Context(), httpx.UserID(c), req)
	})
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if !httpx.Bind(c, "profiles.location", &req) {
		return
	}
	h.respond(c, "profiles.location", func() (MeProfileResponse, error) {
		return h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), req)
	})
}

func (h *Handler) ClearLocation(c *gin.Context) {
	h.respond(c, "profiles.location_clear", func() (MeProfileResponse, error) {
		return h.service.ClearLocation(c.Request.Context(), httpx.UserID(c))
	})
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if !httpx.Bind(c, "profiles.preferences", &req) {
		return
	}
	h.respond(c, "profiles.preferences", func() (MeProfileResponse, error) {
		return h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	})
}

func (h *Handler) UpdatePrivacy(c *gin.Context) {
	var req UpdatePrivacyRequest
	if !httpx.Bind(c, "profiles.privacy", &req) {
		return
	}
	h.respond(c, "profiles.privacy", func() (MeProfileResponse, error) {
		return h.service.UpdatePrivacy(c.Request.Context(), httpx.UserID(c), req)
	})
}

func (h *Handler) GetVisible(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	card, err := h.service.GetVisible(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		httpx.Fail(c, "profiles.get_visible", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, card)
}

func (h *Handler) respond(c *gin.Context, op string, fn func() (MeProfileResponse, error)) {
	resp, err := fn()
	if err != nil {
		httpx.Fail(c, op, err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}
