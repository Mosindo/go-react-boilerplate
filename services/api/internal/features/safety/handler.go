package safety

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "safety.block.bind", "invalid request", err)
		return
	}
	if h.fail(c, "safety.block", h.service.Block(c.Request.Context(), c.GetString("userID"), req.UserID)) {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	if h.fail(c, "safety.unblock", h.service.Unblock(c.Request.Context(), c.GetString("userID"), c.Param("userId"))) {
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Blocked(c *gin.Context) {
	list, err := h.service.Blocked(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		httpx.Internal(c, "safety.blocked", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, BlockedResponse{Blocked: list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "safety.report.bind", "invalid request", err)
		return
	}
	if h.fail(c, "safety.report", h.service.Report(c.Request.Context(), c.GetString("userID"), req)) {
		return
	}
	c.Status(http.StatusCreated)
}

func (h *Handler) fail(c *gin.Context, op string, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, ErrInvalidTarget), errors.Is(err, ErrSelf), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrInvalidText):
		httpx.BadRequest(c, op, err.Error(), err)
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "user not found", nil)
	default:
		httpx.Internal(c, op, err)
	}
	return true
}
