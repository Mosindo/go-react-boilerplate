package profiles

import (
	"errors"
	"net/http"
	"regexp"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func fail(c *gin.Context, op string, err error) {
	var v *ValidationError
	switch {
	case errors.As(err, &v):
		c.JSON(http.StatusBadRequest, gin.H{"error": v.Msg})
	case errors.Is(err, ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func badBody(c *gin.Context, op string, err error) {
	logger.LogHandlerError(c, op, http.StatusBadRequest, err)
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
}

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.Interests(c.Request.Context())
	if err != nil {
		fail(c, "profiles.interests", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"interests": list})
}

func (h *Handler) GetMine(c *gin.Context) {
	p, err := h.service.Own(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		fail(c, "profiles.get", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) PutMine(c *gin.Context) {
	var req ProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badBody(c, "profiles.put.bind", err)
		return
	}
	p, err := h.service.Save(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		fail(c, "profiles.put", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) PutLocation(c *gin.Context) {
	var req LocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badBody(c, "profiles.location.bind", err)
		return
	}
	if err := h.service.SetLocation(c.Request.Context(), c.GetString("userID"), req); err != nil {
		fail(c, "profiles.location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetPreferences(c *gin.Context) {
	p, err := h.service.Preferences(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		fail(c, "profiles.prefs.get", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) PutPreferences(c *gin.Context) {
	var req PreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badBody(c, "profiles.prefs.bind", err)
		return
	}
	p, err := h.service.SavePreferences(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		fail(c, "profiles.prefs.put", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) GetPublic(c *gin.Context) {
	id := c.Param("userId")
	if !uuidRe.MatchString(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	p, err := h.service.Public(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		fail(c, "profiles.public", err)
		return
	}
	c.JSON(http.StatusOK, p)
}
