package safety

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	var ve ValidationError
	switch {
	case errors.As(err, &ve):
		httpx.Fail(c, op, http.StatusUnprocessableEntity, ve.Message, err)
	case errors.Is(err, ErrSelfTarget):
		httpx.Fail(c, op, http.StatusUnprocessableEntity, "you cannot target yourself", err)
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "user not found", err)
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if !httpx.BindJSON(c, "safety.block", &req) {
		return
	}
	if !httpx.IsUUID(req.UserID) {
		httpx.Fail(c, "safety.block", http.StatusNotFound, "user not found", nil)
		return
	}
	if err := h.service.Block(c.Request.Context(), c.GetString("userID"), strings.ToLower(req.UserID)); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "userId")
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
	limit, offset, ok := httpx.Pagination(c, 50, 100)
	if !ok {
		return
	}
	users, err := h.service.ListBlocked(c.Request.Context(), c.GetString("userID"), limit, offset)
	if err != nil {
		h.fail(c, "safety.blocks", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"blocks": users})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, "safety.report", &req) {
		return
	}
	if !httpx.IsUUID(req.UserID) {
		httpx.Fail(c, "safety.report", http.StatusNotFound, "user not found", nil)
		return
	}
	req.UserID = strings.ToLower(req.UserID)
	if err := h.service.Report(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusCreated)
}
