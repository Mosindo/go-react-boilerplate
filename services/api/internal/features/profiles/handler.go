package profiles

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case IsValidation(err):
		httpx.BadRequest(c, err.Error())
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "profile not found")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) Interests(c *gin.Context) {
	items, err := h.service.Interests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": items})
}

func (h *Handler) Own(c *gin.Context) {
	p, err := h.service.Own(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profiles.own", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Save(c *gin.Context) {
	var req ProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request body")
		return
	}
	p, err := h.service.Save(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "profiles.save", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request body")
		return
	}
	p, err := h.service.SavePreferences(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(c, http.StatusConflict, "create your profile first")
			return
		}
		h.fail(c, "profiles.preferences", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) SetLocation(c *gin.Context) {
	var req LocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request body")
		return
	}
	p, err := h.service.SetLocation(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(c, http.StatusConflict, "create your profile first")
			return
		}
		h.fail(c, "profiles.location", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Public(c *gin.Context) {
	id := c.Param("id")
	if !httpx.IsUUID(id) {
		httpx.Error(c, http.StatusNotFound, "profile not found")
		return
	}
	viewer := c.GetString("userID")
	if id == viewer {
		p, err := h.service.Own(c.Request.Context(), viewer)
		if err != nil {
			h.fail(c, "profiles.public.self", err)
			return
		}
		httpx.NoStore(c)
		c.JSON(http.StatusOK, gin.H{"profile": toSelfPublic(p)})
		return
	}
	p, err := h.service.Public(c.Request.Context(), viewer, id)
	if err != nil {
		h.fail(c, "profiles.public", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"profile": p})
}

// toSelfPublic renders the caller's own profile the way others see it, with a
// minimal interest list (labels are resolved client-side from /interests).
func toSelfPublic(p OwnProfile) PublicProfile {
	return PublicProfile{
		ID: p.ID, FirstName: p.FirstName, Age: p.Age, Gender: p.Gender, Bio: p.Bio, City: p.City,
		Interests: []Interest{}, Photos: p.Photos,
	}
}
