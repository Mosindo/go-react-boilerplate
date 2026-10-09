package safety

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
	case errors.Is(err, ErrNotFound):
		httpx.NotFound(c)
	case errors.Is(err, ErrSelf):
		httpx.BadRequest(c, "you cannot target yourself")
	case errors.Is(err, ErrInvalidReport):
		httpx.BadRequest(c, "invalid reason or details (1000 characters max)")
	case errors.Is(err, ErrTooManyReports):
		httpx.Fail(c, http.StatusTooManyRequests, httpx.CodeRateLimited, "daily report limit reached")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) ListBlocks(c *gin.Context) {
	list, err := h.service.ListBlocks(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "safety.blocks", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, BlocksResponse{Blocks: list})
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if !httpx.IsUUID(req.UserID) {
		httpx.NotFound(c)
		return
	}
	if err := h.service.Block(c.Request.Context(), c.GetString("userID"), req.UserID); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	if err := h.service.Unblock(c.Request.Context(), c.GetString("userID"), c.Param("userId")); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if !httpx.IsUUID(req.UserID) {
		httpx.NotFound(c)
		return
	}
	id, err := h.service.Report(c.Request.Context(), c.GetString("userID"), req)
	if err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, ReportResponse{ID: id})
}
