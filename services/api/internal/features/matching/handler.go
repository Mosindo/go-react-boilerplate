package matching

import (
	"errors"
	"net/http"
	"strconv"

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
	case errors.Is(err, ErrInvalidAction), errors.Is(err, ErrInvalidTarget), errors.Is(err, ErrInvalidCursor):
		httpx.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrTargetUnavailable):
		httpx.Error(c, http.StatusNotFound, "not found")
	case errors.Is(err, ErrAlreadySwiped):
		httpx.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrProfileRequired):
		httpx.Error(c, http.StatusConflict, err.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) Discover(c *gin.Context) {
	limit := DefaultDiscoverLimit
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}
	list, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": list})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	result, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) Matches(c *gin.Context) {
	page, err := h.service.Matches(c.Request.Context(), c.GetString("userID"), httpx.Limit(c, httpx.DefaultLimit), c.Query("cursor"))
	if err != nil {
		h.fail(c, "matching.matches", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Unmatch(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Profile(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "id")
	if !ok {
		return
	}
	p, err := h.service.Profile(c.Request.Context(), c.GetString("userID"), id)
	if err != nil {
		h.fail(c, "matching.profile", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, p)
}
