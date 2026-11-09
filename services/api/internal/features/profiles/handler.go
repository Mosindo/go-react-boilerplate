package profiles

import (
	"errors"
	"net/http"
	"regexp"

	"example.com/api/internal/platform/geo"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

func validCoords(lat, lon float64) bool { return geo.ValidCoords(lat, lon) }

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsUUID reports whether s is a canonical UUID; handlers use it to reject malformed ids with 400/404.
func IsUUID(s string) bool { return uuidRe.MatchString(s) }

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrUnknownInterest):
		httpx.Error(c, http.StatusBadRequest, "unknown interest id")
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "profile not found")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) GetMine(c *gin.Context) {
	p, err := h.service.GetOwn(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "profiles.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) SaveMine(c *gin.Context) {
	var in ProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.Save(c.Request.Context(), c.GetString("userID"), in)
	if err != nil {
		h.fail(c, "profiles.save", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) SetLocation(c *gin.Context) {
	var in LocationInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.SetLocation(c.Request.Context(), c.GetString("userID"), in); err != nil {
		h.fail(c, "profiles.location", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) SavePreferences(c *gin.Context) {
	var in Preferences
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.SavePreferences(c.Request.Context(), c.GetString("userID"), in)
	if err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": list})
}

func (h *Handler) GetPublic(c *gin.Context) {
	id := c.Param("id")
	if !IsUUID(id) {
		httpx.Error(c, http.StatusNotFound, "profile not found")
		return
	}
	p, err := h.service.GetPublic(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		h.fail(c, "profiles.public", err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, p)
}
