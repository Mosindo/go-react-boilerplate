package safety

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrSelf), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrDetailsLong):
		httpx.BadRequest(c, err.Error())
	case errors.Is(err, ErrUnknownUser):
		httpx.Fail(c, http.StatusNotFound, "not_found", err.Error())
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
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
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), id); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocked(c *gin.Context) {
	list, err := h.service.ListBlocked(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Internal(c, "safety.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocks": list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "invalid request")
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), req); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusCreated)
}
