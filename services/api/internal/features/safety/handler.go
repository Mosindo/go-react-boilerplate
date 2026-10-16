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
	case errors.Is(err, ErrInvalidUser), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrDetailsLength):
		httpx.Fail(c, op, http.StatusBadRequest, err.Error(), err)
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "user not found", err)
	default:
		httpx.Fail(c, op, http.StatusInternalServerError, "internal error", err)
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
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
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		httpx.Fail(c, "safety.unblock.id", http.StatusBadRequest, "invalid user id", errors.New("invalid user id"))
		return
	}
	if err := h.service.Unblock(c.Request.Context(), c.GetString("userID"), targetID); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Blocked(c *gin.Context) {
	list, err := h.service.Blocked(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "safety.blocked", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"blocks": list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, "safety.report.bind", http.StatusBadRequest, "invalid request", err)
		return
	}
	if err := h.service.Report(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusAccepted)
}
