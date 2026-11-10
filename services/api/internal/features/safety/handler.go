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

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "userId (uuid) is required")
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
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

func (h *Handler) ListBlocked(c *gin.Context) {
	users, err := h.service.ListBlocked(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		h.fail(c, "safety.list_blocked", err)
		return
	}
	out := make([]BlockedUserResponse, 0, len(users))
	for _, u := range users {
		out = append(out, BlockedUserResponse{UserID: u.UserID, FirstName: u.FirstName, BlockedAt: u.CreatedAt})
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, gin.H{"blocked": out})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid_request", "userId (uuid) and reason are required")
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), req.UserID, req.Reason, req.Details); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "reported"})
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrSelf), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrDetailsLong):
		httpx.BadRequest(c, "validation_failed", err.Error())
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}
