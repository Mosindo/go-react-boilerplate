package safety

import (
	"errors"
	"net/http"
	"strings"

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
	case errors.Is(err, ErrSelfAction):
		httpx.BadRequest(c, "self_action", err.Error())
	case errors.Is(err, ErrInvalidReason):
		httpx.BadRequest(c, "invalid_reason", err.Error())
	case errors.Is(err, ErrDetailsLength):
		httpx.BadRequest(c, "details_too_long", err.Error())
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", "user not found", operation, nil)
	default:
		httpx.Internal(c, operation, err)
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "userId required")
		return
	}
	target := strings.ToLower(strings.TrimSpace(req.UserID))
	if !httpx.IsUUID(target) {
		httpx.BadRequest(c, "invalid_id", "invalid userId")
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
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), target); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	res, err := h.service.ListBlocks(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "safety.blocks", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "userId and reason are required")
		return
	}
	if !httpx.IsUUID(strings.TrimSpace(req.UserID)) {
		httpx.BadRequest(c, "invalid_id", "invalid userId")
		return
	}
	res, err := h.service.Report(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, res)
}
