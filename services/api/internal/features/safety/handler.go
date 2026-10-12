package safety

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpx.Fail(c, http.StatusUnprocessableEntity, "invalid request")
	case errors.Is(err, ErrNotFound):
		httpx.Fail(c, http.StatusNotFound, "user not found")
	default:
		httpx.Internal(c, op, err)
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if !httpx.IsUUID(req.UserID) {
		httpx.Fail(c, http.StatusUnprocessableEntity, "invalid user")
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

func (h *Handler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "safety.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if !httpx.IsUUID(req.UserID) {
		httpx.Fail(c, http.StatusUnprocessableEntity, "invalid user")
		return
	}
	req.UserID = strings.ToLower(req.UserID)
	if err := h.service.Report(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusNoContent)
}
