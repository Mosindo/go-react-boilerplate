package safety

import (
	"errors"
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

func (h *Handler) fail(c *gin.Context, operation string, err error) {
	switch {
	case errors.Is(err, ErrSelfAction), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrDetailsLength):
		httpx.Fail(c, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusNotFound, "not_found", err.Error())
	default:
		httpx.FailInternal(c, operation, err)
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if !httpx.BindJSON(c, "safety.block", &req) {
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), req.UserID); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	userID, ok := httpx.PathUUID(c, "userId")
	if !ok {
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), userID); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocked(c *gin.Context) {
	limit, ok := httpx.QueryLimit(c, 50, 100)
	if !ok {
		return
	}
	offset, ok := httpx.QueryOffset(c)
	if !ok {
		return
	}
	list, err := h.service.ListBlocked(c.Request.Context(), httpx.UserID(c), limit, offset)
	if err != nil {
		h.fail(c, "safety.list_blocked", err)
		return
	}
	payload := make([]BlockedUserResponse, 0, len(list))
	for _, b := range list {
		payload = append(payload, BlockedUserResponse(b))
	}
	c.JSON(http.StatusOK, gin.H{"blocked": payload})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if !httpx.BindJSON(c, "safety.report", &req) {
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), req.UserID, req.Reason, req.Details); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "received"})
}
