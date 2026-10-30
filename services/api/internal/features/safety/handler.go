package safety

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "user not found", nil)
	case errors.Is(err, ErrTooManyReports):
		httpx.Fail(c, op, http.StatusTooManyRequests, "daily report limit reached", nil)
	default:
		httpx.WriteError(c, op, err)
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil || !httpx.IsUUID(req.UserID) {
		httpx.Fail(c, "safety.block.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := h.service.Block(c.Request.Context(), c.GetString("userID"), req.UserID); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	id, ok := httpx.ParamUUID(c, "userId")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocked(c *gin.Context) {
	list, err := h.service.ListBlocked(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "safety.blocks", err)
		return
	}
	c.JSON(http.StatusOK, BlockedUsersResponse{Blocked: list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil || !httpx.IsUUID(req.UserID) {
		httpx.Fail(c, "safety.report.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := h.service.Report(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusCreated)
}
