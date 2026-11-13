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

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, ErrSelfAction), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrDetailsTooLong):
		httpx.FailCode(c, operation, http.StatusBadRequest, "validation", err.Error(), err)
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, operation, http.StatusNotFound, "user not found", err)
	default:
		httpx.Fail(c, operation, http.StatusInternalServerError, "could not process request", err)
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "safety.block.bind", "userId is required", err)
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	target, ok := httpx.UUIDParam(c, "userId", "safety.unblock.validate")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), target); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocked(c *gin.Context) {
	items, err := h.service.ListBlocked(c.Request.Context(), httpx.UserID(c),
		httpx.Limit(c.Query("limit"), 50, 100), httpx.Offset(c.Query("offset"), 10_000))
	if err != nil {
		h.fail(c, "safety.blocks", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocks": items})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "safety.report.bind", "userId and reason are required", err)
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), req.UserID, req.Reason, req.Details); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusCreated)
}
