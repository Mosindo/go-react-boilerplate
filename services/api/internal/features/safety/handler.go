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

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_user_id", "a valid userId is required")
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), strings.ToLower(req.UserID)); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	id := strings.ToLower(strings.TrimSpace(c.Param("userId")))
	if !httpx.IsUUID(id) {
		httpx.BadRequest(c, "invalid_user_id", "invalid user id")
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	list, err := h.service.ListBlocks(c.Request.Context(), httpx.UserID(c),
		httpx.Limit(c.Query("limit"), 50, 100), httpx.Offset(c.Query("offset"), 10_000))
	if err != nil {
		h.fail(c, "safety.list_blocks", err)
		return
	}
	c.JSON(http.StatusOK, BlocksResponse{Blocks: list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_report", "userId and reason are required")
		return
	}
	req.UserID = strings.ToLower(req.UserID)
	res, err := h.service.Report(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrSelfAction):
		httpx.Fail(c, op, http.StatusBadRequest, "self_action", err.Error(), nil)
	case errors.Is(err, ErrInvalidReason):
		httpx.Fail(c, op, http.StatusBadRequest, "invalid_reason", err.Error(), nil)
	case errors.Is(err, ErrDetailsTooLong):
		httpx.Fail(c, op, http.StatusBadRequest, "details_too_long", err.Error(), nil)
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "user_not_found", err.Error(), nil)
	default:
		httpx.Internal(c, op, err)
	}
}
