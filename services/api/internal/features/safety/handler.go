package safety

import (
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

func targetID(c *gin.Context, operation string) (string, bool) {
	id := strings.ToLower(c.Param("userId"))
	if !httpx.IsUUID(id) {
		httpx.Fail(c, operation, ErrUserNotFound)
		return "", false
	}
	return id, true
}

func (h *Handler) Block(c *gin.Context) {
	id, ok := targetID(c, "safety.block")
	if !ok {
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	id, ok := targetID(c, "safety.unblock")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocked(c *gin.Context) {
	limit, offset, ok := httpx.Page(c, 50, maxBlocksLimit)
	if !ok {
		return
	}
	list, err := h.service.ListBlocked(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		httpx.Fail(c, "safety.list_blocked", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, BlockedResponse{Blocked: list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.Bind(c, &req) {
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), req); err != nil {
		httpx.Fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusAccepted)
}
