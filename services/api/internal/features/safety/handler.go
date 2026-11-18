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

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", "user not found", operation, err)
	case errors.Is(err, ErrSelfAction), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrDetailsLength):
		httpx.Fail(c, http.StatusBadRequest, "validation_failed", err.Error(), operation, err)
	default:
		httpx.Fail(c, http.StatusInternalServerError, "internal", "could not process request", operation, err)
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil || !httpx.IsUUID(req.UserID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid user id", "safety.block.bind", err)
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), httpx.NormalizeUUID(req.UserID)); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	target := httpx.NormalizeUUID(strings.TrimSpace(c.Param("userId")))
	if !httpx.IsUUID(target) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_id", "invalid user id", "safety.unblock.validate", nil)
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), target); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	limit := httpx.Limit(c.Query("limit"), 50, 100)
	offset := httpx.Offset(c.Query("offset"), 10_000)
	blocks, err := h.service.ListBlocks(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		h.fail(c, "safety.blocks", err)
		return
	}
	c.JSON(http.StatusOK, BlocksResponse{Blocks: blocks})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil || !httpx.IsUUID(req.UserID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "invalid request", "safety.report.bind", err)
		return
	}
	target := httpx.NormalizeUUID(req.UserID)
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), target, req.Reason, req.Details); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	logger.LogHandlerEvent(c, "safety.report.created", http.StatusCreated, map[string]string{"reported_user_id": target, "reason": req.Reason})
	c.JSON(http.StatusCreated, gin.H{"status": "reported"})
}
