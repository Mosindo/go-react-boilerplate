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

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.InvalidRequest(c)
		return
	}
	target := strings.ToLower(strings.TrimSpace(req.UserID))
	if !httpx.IsUUID(target) {
		httpx.Error(c, http.StatusBadRequest, "invalid_user_id", "invalid user id")
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), target); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	target, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		httpx.Error(c, http.StatusBadRequest, "invalid_user_id", "invalid user id")
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), target); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	limit := httpx.ParseLimit(c.Query("limit"), 50, 100)
	offset := httpx.ParseOffset(c.Query("offset"))
	blocks, err := h.service.ListBlocks(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		h.fail(c, "safety.blocks", err)
		return
	}
	c.JSON(http.StatusOK, BlocksResponse{Blocks: blocks})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.InvalidRequest(c)
		return
	}
	target := strings.ToLower(strings.TrimSpace(req.UserID))
	if !httpx.IsUUID(target) {
		httpx.Error(c, http.StatusBadRequest, "invalid_user_id", "invalid user id")
		return
	}
	resp, err := h.service.Report(c.Request.Context(), httpx.UserID(c), target, req.Reason, req.Details)
	if err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, resp)
	logger.LogHandlerEvent(c, "safety.report.created", http.StatusCreated, map[string]string{"report_id": resp.ID})
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, ErrSelfAction), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrDetailsLength):
		status = http.StatusBadRequest
		httpx.Error(c, status, "validation_error", err.Error())
	case errors.Is(err, ErrUserNotFound):
		status = http.StatusNotFound
		httpx.Error(c, status, "user_not_found", "user not found")
	default:
		httpx.Internal(c, "could not process request")
	}
	logger.LogHandlerError(c, op, status, err)
}
