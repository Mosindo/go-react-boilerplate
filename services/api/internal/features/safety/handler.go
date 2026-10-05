package safety

import (
	"errors"
	"net/http"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrSelfAction):
		httpx.Fail(c, op, http.StatusBadRequest, "self_action", err.Error(), err)
	case errors.Is(err, ErrInvalidReason):
		httpx.Fail(c, op, http.StatusBadRequest, "invalid_reason", err.Error(), err)
	case errors.Is(err, ErrDetailsLength):
		httpx.Fail(c, op, http.StatusBadRequest, "invalid_details", err.Error(), err)
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, op, http.StatusNotFound, "not_found", "user not found", err)
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
		httpx.Fail(c, "safety.block", http.StatusBadRequest, "invalid_id", "invalid userId", nil)
		return
	}
	if err := h.service.Block(c.Request.Context(), c.GetString("userID"), normalizeID(req.UserID)); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "safety.unblock", "userId")
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
	blocks, err := h.service.ListBlocks(c.Request.Context(), c.GetString("userID"),
		httpx.Limit(c.Query("limit"), 50, 100), httpx.Offset(c.Query("offset")))
	if err != nil {
		h.fail(c, "safety.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, BlocksResponse{Blocks: blocks})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, "safety.report", &req) {
		return
	}
	if !httpx.IsUUID(req.UserID) {
		httpx.Fail(c, "safety.report", http.StatusBadRequest, "invalid_id", "invalid userId", nil)
		return
	}
	req.UserID = normalizeID(req.UserID)
	if err := h.service.Report(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusCreated)
}
