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
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
		httpx.Respond(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), c.Param("userId")); err != nil {
		httpx.Respond(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	res, err := h.service.ListBlocks(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Respond(c, "safety.blocks", err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), req); err != nil {
		httpx.Respond(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{})
}
