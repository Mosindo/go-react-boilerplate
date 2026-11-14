package profiles

import (
	"errors"
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

func (h *Handler) fail(c *gin.Context, op string, err error) {
	var verr *ValidationError
	switch {
	case errors.As(err, &verr):
		httpx.BadRequest(c, verr.Code, verr.Message)
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusConflict, "profile_required", "complétez d'abord votre profil")
	case errors.Is(err, ErrForbidden):
		httpx.Fail(c, http.StatusNotFound, "profile_not_found", "profil introuvable")
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Interests(c *gin.Context) {
	interests, err := h.service.Interests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": interests})
}

func (h *Handler) Mine(c *gin.Context) {
	resp, err := h.service.Mine(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "profiles.mine", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateProfileRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	resp, err := h.service.UpdateProfile(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "profiles.update", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	prefs, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (h *Handler) UpdateLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpdateLocation(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "profiles.location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)); err != nil {
		h.fail(c, "profiles.location_clear", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) UpdatePrivacy(c *gin.Context) {
	var req UpdatePrivacyRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.UpdatePrivacy(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "profiles.privacy", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) View(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	profile, err := h.service.View(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		h.fail(c, "profiles.view", err)
		return
	}
	c.JSON(http.StatusOK, profile)
}
