package profiles

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	if v, ok := httpx.AsValidation(err); ok {
		httpx.Error(c, http.StatusBadRequest, v.Message)
		return
	}
	switch {
	case errors.Is(err, ErrProfileNotFound), errors.Is(err, ErrNotViewable):
		httpx.Error(c, http.StatusNotFound, "profile not found")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) GetOwn(c *gin.Context) {
	p, err := h.service.GetOwn(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "profiles.get_own", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.Upsert(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "profiles.upsert", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) GetPublic(c *gin.Context) {
	targetID, ok := httpx.ParamUUID(c, "userId")
	if !ok {
		return
	}
	p, err := h.service.GetPublic(c.Request.Context(), httpx.UserID(c), targetID)
	if err != nil {
		h.fail(c, "profiles.get_public", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, p)
}

func (h *Handler) GetPreferences(c *gin.Context) {
	p, err := h.service.GetPreferences(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "profiles.get_preferences", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) UpdatePreferences(c *gin.Context) {
	var req UpdatePreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.UpdatePreferences(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "profiles.update_preferences", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) SetLocation(c *gin.Context) {
	var req UpdateLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.SetLocation(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "profiles.set_location", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) ClearLocation(c *gin.Context) {
	if err := h.service.ClearLocation(c.Request.Context(), httpx.UserID(c)); err != nil {
		h.fail(c, "profiles.clear_location", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) ListInterests(c *gin.Context) {
	list, err := h.service.ListInterests(c.Request.Context())
	if err != nil {
		h.fail(c, "profiles.list_interests", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"interests": list})
}
