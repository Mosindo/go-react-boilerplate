package safety

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if !httpx.Bind(c, "safety.block", &req) {
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
		httpx.Fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	targetID, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), targetID); err != nil {
		httpx.Fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	resp, err := h.service.ListBlocks(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "safety.list_blocks", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.Bind(c, "safety.report", &req) {
		return
	}
	resp, err := h.service.Report(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}
