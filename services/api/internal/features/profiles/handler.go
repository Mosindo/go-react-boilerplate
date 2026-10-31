package profiles

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) GetOwn(c *gin.Context) {
	p, err := h.service.GetOwn(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Internal(c, "profiles.get_own", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Update(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "profiles.update.bind", "invalid request", err)
		return
	}
	p, err := h.service.Update(c.Request.Context(), c.GetString("userID"), req)
	if h.writeErr(c, "profiles.update", err) {
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "profiles.prefs.bind", "invalid request", err)
		return
	}
	p, err := h.service.UpdatePreferences(c.Request.Context(), c.GetString("userID"), req)
	if h.writeErr(c, "profiles.prefs", err) {
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Internal(c, "profiles.interests", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"interests": list})
}

func (h *Handler) Get(c *gin.Context) {
	card, err := h.service.Card(c.Request.Context(), c.GetString("userID"), c.Param("id"))
	switch {
	case err == nil:
		httpx.NoStore(c)
		c.JSON(http.StatusOK, card)
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, "profiles.get", http.StatusNotFound, "profile not found", nil)
	default:
		httpx.Internal(c, "profiles.get", err)
	}
}

// writeErr maps domain errors to responses and reports whether it handled err.
func (h *Handler) writeErr(c *gin.Context, op string, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidGender), errors.Is(err, ErrInvalidBio),
		errors.Is(err, ErrInvalidCity), errors.Is(err, ErrInvalidLocation), errors.Is(err, ErrTooManyInterests),
		errors.Is(err, ErrInvalidPrefs), errors.Is(err, ErrUnknownInterests):
		httpx.BadRequest(c, op, err.Error(), err)
	case errors.Is(err, ErrProfileMissing):
		httpx.Fail(c, op, http.StatusConflict, err.Error(), err)
	default:
		httpx.Internal(c, op, err)
	}
	return true
}
