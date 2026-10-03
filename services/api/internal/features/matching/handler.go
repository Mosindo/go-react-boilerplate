package matching

import (
	"errors"
	"net/http"
	"strconv"

	"example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/logger"
	"example.com/api/internal/platform/validate"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		c.JSON(http.StatusBadRequest, gin.H{"error": ve.Message})
	case errors.Is(err, ErrProfileIncomplete):
		c.JSON(http.StatusConflict, gin.H{"error": ErrProfileIncomplete.Error(), "code": "profile_incomplete"})
	case errors.Is(err, ErrNotEligible), errors.Is(err, profiles.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
	}
}

func (h *Handler) Discover(c *gin.Context) {
	limit := DefaultDiscoverLimit
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxDiscoverLimit {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be between 1 and 20"})
			return
		}
		limit = n
	}
	items, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	if items == nil {
		items = []profiles.PublicProfile{}
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, DiscoverResponse{Profiles: items})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if !validate.IsUUID(req.TargetID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}
	resp, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req.TargetID, req.Action)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}
