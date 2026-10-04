package safety

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	blocks, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "blocks.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocks": blocks})
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
		httpx.Fail(c, "blocks.create", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) Unblock(c *gin.Context) {
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), c.Param("userId")); err != nil {
		httpx.Fail(c, "blocks.delete", err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	id, err := h.service.Report(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "reports.create", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}
