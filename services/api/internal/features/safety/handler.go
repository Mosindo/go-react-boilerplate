package safety

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Blocks(c *gin.Context) {
	items, err := h.service.Blocked(c.Request.Context(), httpx.UserID(c))
	if err != nil {
		httpx.Respond(c, "safety.blocks", err)
		return
	}
	c.JSON(http.StatusOK, httpx.Items(items))
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if !httpx.IsUUID(req.UserID) {
		httpx.Fail(c, http.StatusBadRequest, "validation_error", "userId must be a UUID")
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
		httpx.Respond(c, "safety.block", err)
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
		httpx.Respond(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, &req) {
		return
	}
	if !httpx.IsUUID(req.TargetUserID) {
		httpx.Fail(c, http.StatusBadRequest, "validation_error", "targetUserId must be a UUID")
		return
	}
	id, err := h.service.Report(c.Request.Context(), httpx.UserID(c), req)
	if err != nil {
		httpx.Respond(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}
