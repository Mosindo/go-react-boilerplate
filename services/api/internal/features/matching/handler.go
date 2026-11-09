package matching

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
	switch {
	case errors.Is(err, ErrSelf), errors.Is(err, ErrInvalidAction):
		httpx.Error(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrIncompleteProfile):
		httpx.Error(c, http.StatusConflict, err.Error())
	case errors.Is(err, ErrTargetUnavailable), errors.Is(err, ErrMatchNotFound):
		httpx.Error(c, http.StatusNotFound, err.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Error(c, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) Discover(c *gin.Context) {
	limit := httpx.IntQuery(c, "limit", DefaultPageSize, 1, MaxPageSize)
	cards, err := h.service.Discover(c.Request.Context(), c.GetString("userID"), limit)
	if err != nil {
		h.fail(c, "matching.discover", err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, gin.H{"profiles": cards})
}

func (h *Handler) Swipe(c *gin.Context) {
	var req SwipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	res, err := h.service.Swipe(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "matching.swipe", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) ResetPasses(c *gin.Context) {
	n, err := h.service.ResetPasses(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "matching.reset", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"reset": n})
}

func (h *Handler) Unmatch(c *gin.Context) {
	if err := h.service.Unmatch(c.Request.Context(), c.GetString("userID"), c.Param("id")); err != nil {
		h.fail(c, "matching.unmatch", err)
		return
	}
	c.Status(http.StatusNoContent)
}
