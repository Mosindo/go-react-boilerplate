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

func (h *Handler) Get(c *gin.Context) {
	p, err := h.service.Get(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		writeError(c, "profiles.get", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Upsert(c *gin.Context) {
	var req UpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid profile data")
		return
	}
	p, err := h.service.Upsert(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		writeError(c, "profiles.upsert", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}

func (h *Handler) UpdateSettings(c *gin.Context) {
	var req SettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil || (req.IsVisible == nil && req.ShowDistance == nil) {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	p, err := h.service.UpdateSettings(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		writeError(c, "profiles.settings", err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) Interests(c *gin.Context) {
	list, err := h.service.Interests(c.Request.Context())
	if err != nil {
		writeError(c, "profiles.interests", err)
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.JSON(http.StatusOK, InterestsResponse{Interests: list})
}

func writeError(c *gin.Context, op string, err error) {
	status, message := http.StatusInternalServerError, "something went wrong"
	switch {
	case errors.Is(err, ErrNotFound):
		status, message = http.StatusNotFound, "profile not found"
	case errors.Is(err, ErrInvalidName), errors.Is(err, ErrInvalidBirthDate), errors.Is(err, ErrTooOld),
		errors.Is(err, ErrBioTooLong), errors.Is(err, ErrCityTooLong), errors.Is(err, ErrUnknownInterest),
		errors.Is(err, ErrTooManyInterests), errors.Is(err, ErrInvalidPrefs):
		status, message = http.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, ErrUnderage):
		status, message = http.StatusForbidden, err.Error()
	}
	logger.LogHandlerError(c, op, status, err)
	httpx.Error(c, status, message)
}
