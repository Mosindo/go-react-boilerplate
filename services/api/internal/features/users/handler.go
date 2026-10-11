package users

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Respond(c, "users.interests", err)
		return
	}
	c.Header("Cache-Control", "private, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": list})
}

func (h *Handler) GetOwnProfile(c *gin.Context) {
	p, err := h.service.Own(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Respond(c, "users.own", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) PutProfile(c *gin.Context) {
	var in ProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "invalid profile payload")
		return
	}
	p, err := h.service.SaveProfile(c.Request.Context(), httpx.UserID(c), in)
	if err != nil {
		httpx.Respond(c, "users.save_profile", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) PutLocation(c *gin.Context) {
	var in LocationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "latitude and longitude are required")
		return
	}
	if err := h.service.SetLocation(c.Request.Context(), httpx.UserID(c), *in.Latitude, *in.Longitude); err != nil {
		httpx.Respond(c, "users.location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) DeleteLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Respond(c, "users.clear_location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetPreferences(c *gin.Context) {
	p, err := h.service.Preferences(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Respond(c, "users.preferences", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) PutPreferences(c *gin.Context) {
	var in Preferences
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "invalid preferences payload")
		return
	}
	p, err := h.service.SavePreferences(c.Request.Context(), httpx.UserID(c), in)
	if err != nil {
		httpx.Respond(c, "users.save_preferences", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) GetPrivacy(c *gin.Context) {
	p, err := h.service.Privacy(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Respond(c, "users.privacy", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) PutPrivacy(c *gin.Context) {
	var in Privacy
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "discoverable and showDistance are required")
		return
	}
	if err := h.service.SavePrivacy(c.Request.Context(), httpx.UserID(c), *in.Discoverable, *in.ShowDistance); err != nil {
		httpx.Respond(c, "users.save_privacy", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"discoverable": *in.Discoverable, "showDistance": *in.ShowDistance})
}

func (h *Handler) GetProfile(c *gin.Context) {
	target, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	p, err := h.service.Profile(c.Request.Context(), httpx.UserID(c), target)
	if err != nil {
		httpx.Respond(c, "users.profile", err)
		return
	}
	c.Header("Cache-Control", "private, no-cache")
	c.JSON(http.StatusOK, p)
}
