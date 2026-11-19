package safety

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "userId is required")
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), strings.ToLower(req.UserID)); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Blocks(c *gin.Context) {
	items, err := h.service.Blocks(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "safety.blocks", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocks": items})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "userId and reason are required")
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	logger.LogHandlerEvent(c, "safety.report.created", http.StatusCreated, map[string]string{"reason": req.Reason})
	c.JSON(http.StatusCreated, gin.H{"status": "received"})
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusNotFound, "user_not_found", err.Error())
	case errors.Is(err, ErrSelf), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrInvalidDetail):
		httpx.BadRequest(c, err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}
