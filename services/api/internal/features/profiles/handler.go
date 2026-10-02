package profiles

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	var ve *ValidationError
	switch {
	case errors.Is(err, ErrUnderage):
		httpx.Fail(c, http.StatusUnprocessableEntity, "underage", err.Error())
	case errors.Is(err, ErrBirthDateLocked):
		httpx.Fail(c, http.StatusConflict, "birth_date_locked", err.Error())
	case errors.As(err, &ve):
		httpx.Fail(c, http.StatusBadRequest, "invalid_"+ve.Field, ve.Message)
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", "create your profile first")
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Get(c *gin.Context) {
	st, err := h.service.Status(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "profiles.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, st)
}

func (h *Handler) Put(c *gin.Context) {
	var in ProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if err := h.service.Save(c.Request.Context(), httpx.UserID(c), in); err != nil {
		h.fail(c, "profiles.put", err)
		return
	}
	h.Get(c)
}

func (h *Handler) PutPreferences(c *gin.Context) {
	var in Preferences
	if err := c.ShouldBindJSON(&in); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if _, err := h.service.SavePreferences(c.Request.Context(), httpx.UserID(c), in); err != nil {
		h.fail(c, "profiles.preferences", err)
		return
	}
	h.Get(c)
}

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.Interests(c.Request.Context())
	if err != nil {
		httpx.Internal(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, gin.H{"interests": list})
}
