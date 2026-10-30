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

func (h *Handler) GetOwn(c *gin.Context) {
	profile, err := h.service.GetOwn(c.Request.Context(), c.GetString("userID"))
	if errors.Is(err, ErrNotFound) {
		httpx.Fail(c, "profiles.get", http.StatusNotFound, "profile not found", nil)
		return
	}
	if httpx.WriteError(c, "profiles.get", err) {
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) Save(c *gin.Context) {
	var req SaveProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "profiles.save.bind", err)
		return
	}
	profile, err := h.service.Save(c.Request.Context(), c.GetString("userID"), req)
	if httpx.WriteError(c, "profiles.save", err) {
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, profile)
}

func (h *Handler) GetPreferences(c *gin.Context) {
	prefs, err := h.service.GetPreferences(c.Request.Context(), c.GetString("userID"))
	if errors.Is(err, ErrNotFound) {
		httpx.Fail(c, "profiles.prefs.get", http.StatusNotFound, "profile not found", nil)
		return
	}
	if httpx.WriteError(c, "profiles.prefs.get", err) {
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var req SavePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "profiles.prefs.bind", err)
		return
	}
	prefs, err := h.service.SavePreferences(c.Request.Context(), c.GetString("userID"), req)
	if errors.Is(err, ErrNotFound) {
		httpx.Fail(c, "profiles.prefs.save", http.StatusNotFound, "create your profile first", nil)
		return
	}
	if httpx.WriteError(c, "profiles.prefs.save", err) {
		return
	}
	c.JSON(http.StatusOK, prefs)
}

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.Interests(c.Request.Context())
	if httpx.WriteError(c, "profiles.interests", err) {
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": list})
}

func (h *Handler) GetPublic(c *gin.Context) {
	targetID, ok := httpx.ParamUUID(c, "id")
	if !ok {
		return
	}
	profile, err := h.service.GetPublic(c.Request.Context(), c.GetString("userID"), targetID)
	if errors.Is(err, ErrNotFound) {
		httpx.Fail(c, "profiles.public", http.StatusNotFound, "profile not found", nil)
		return
	}
	if httpx.WriteError(c, "profiles.public", err) {
		return
	}
	c.JSON(http.StatusOK, profile)
}
