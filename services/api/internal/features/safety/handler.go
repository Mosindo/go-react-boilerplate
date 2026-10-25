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

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	var ve ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.BadRequest(c, ve.Message)
	case IsNotFound(err):
		httpx.NotFound(c)
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil || !httpx.IsUUID(strings.ToLower(req.UserID)) {
		httpx.BadRequest(c, "userId must be a UUID")
		return
	}
	if err := h.service.Block(c.Request.Context(), c.GetString("userID"), strings.ToLower(req.UserID)); err != nil {
		h.fail(c, "blocks.create", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	target, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), c.GetString("userID"), target); err != nil {
		h.fail(c, "blocks.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocked(c *gin.Context) {
	limit, err := httpx.Limit(c, httpx.DefaultLimit, httpx.MaxLimit)
	if err != nil {
		httpx.BadRequest(c, err.Error())
		return
	}
	cur, ok := httpx.QueryCursor(c)
	if !ok {
		return
	}
	page, err := h.service.ListBlocked(c.Request.Context(), c.GetString("userID"), limit, cur)
	if err != nil {
		h.fail(c, "blocks.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, page)
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil || !httpx.IsUUID(strings.ToLower(req.UserID)) {
		httpx.BadRequest(c, "userId must be a UUID and reason is required")
		return
	}
	id, err := h.service.Report(c.Request.Context(), c.GetString("userID"), strings.ToLower(req.UserID), req.Reason, req.Details)
	if err != nil {
		h.fail(c, "reports.create", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}
