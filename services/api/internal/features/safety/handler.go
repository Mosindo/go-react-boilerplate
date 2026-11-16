package safety

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status, code, msg := http.StatusInternalServerError, "internal", "something went wrong"
	switch {
	case errors.Is(err, ErrSelf):
		status, code, msg = http.StatusBadRequest, "invalid_target", err.Error()
	case errors.Is(err, ErrUserNotFound), errors.Is(err, ErrNotBlocked):
		status, code, msg = http.StatusNotFound, "not_found", err.Error()
	}
	logger.LogHandlerError(c, op, status, err)
	httpx.Fail(c, status, code, msg)
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "userId is required")
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	userID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), userID); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	items, err := h.service.ListBlocks(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "safety.blocks", err)
		return
	}
	c.JSON(http.StatusOK, BlocksResponse{Blocks: items})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "invalid_request", "userId and a valid reason are required")
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), req.UserID, req.Reason, req.Details); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusNoContent)
}
