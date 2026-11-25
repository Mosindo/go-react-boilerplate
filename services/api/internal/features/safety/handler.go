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

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if !httpx.BindJSON(c, "safety.block", &req) {
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), strings.ToLower(req.UserID)); err != nil {
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
	list, err := h.service.ListBlocks(c.Request.Context(), httpx.UserID(c), httpx.Limit(c, 50, 100), httpx.Offset(c))
	if err != nil {
		httpx.Fail(c, "safety.blocks", err)
		return
	}
	httpx.NoStore(c)
	c.JSON(http.StatusOK, BlocksResponse{Blocks: list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, "safety.report", &req) {
		return
	}
	req.UserID = strings.ToLower(req.UserID)
	resp, err := h.service.Report(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, resp)
}
