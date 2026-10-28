package safety

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
		httpx.Fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "userId")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), id); err != nil {
		httpx.Fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) List(c *gin.Context) {
	list, err := h.service.List(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Fail(c, "safety.list", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"blocks": list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	id, err := h.service.Report(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}
