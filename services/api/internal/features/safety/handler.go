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
	switch {
	case errors.Is(err, ErrInvalidTarget), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrDetailsTooLong):
		httpx.BadRequest(c, err.Error())
	case errors.Is(err, ErrUserNotFound):
		httpx.Error(c, http.StatusNotFound, "user not found")
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Internal(c)
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil || !httpx.IsUUID(req.UserID) {
		httpx.BadRequest(c, "userId must be a valid id")
		return
	}
	if err := h.service.Block(c.Request.Context(), c.GetString("userID"), req.UserID); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	id := c.Param("userId")
	if !httpx.IsUUID(id) {
		httpx.BadRequest(c, "userId must be a valid id")
		return
	}
	if err := h.service.Unblock(c.Request.Context(), c.GetString("userID"), id); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Blocked(c *gin.Context) {
	items, err := h.service.Blocked(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "safety.blocked", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"blocks": items})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil || !httpx.IsUUID(req.UserID) {
		httpx.BadRequest(c, "userId and reason are required")
		return
	}
	if err := h.service.Report(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "received"})
}
