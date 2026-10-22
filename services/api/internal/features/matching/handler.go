package matching

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"example.com/api/internal/features/profiles"
	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func writeError(c *gin.Context, op string, err error) {
	var validation *platformerrors.ValidationError
	status := http.StatusInternalServerError
	body := gin.H{"error": "internal error"}
	switch {
	case errors.As(err, &validation):
		status, body = http.StatusBadRequest, gin.H{"error": validation.Message, "field": validation.Field}
	case errors.Is(err, ErrProfileIncomplete):
		status, body = http.StatusConflict, gin.H{"error": "complete your profile first", "code": "profile_incomplete"}
	case errors.Is(err, ErrNotEligible), errors.Is(err, ErrMatchNotFound):
		status, body = http.StatusNotFound, gin.H{"error": "not found"}
	}
	logger.LogHandlerError(c, op, status, err)
	c.JSON(status, body)
}

func (h *Handler) Discover(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	list, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		writeError(c, "discover.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, DiscoverResponse{Profiles: list})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	result, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		writeError(c, "discover.swipe", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ListMatches(c *gin.Context) {
	var before *time.Time
	if raw := c.Query("cursor"); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cursor"})
			return
		}
		before = &t
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	resp, err := h.service.ListMatches(c.Request.Context(), c.GetString("userID"), before, limit)
	if err != nil {
		writeError(c, "matches.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Unmatch(c *gin.Context) {
	id := c.Param("id")
	if !profiles.IsUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), id); err != nil {
		writeError(c, "matches.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}
