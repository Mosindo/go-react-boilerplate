package safety

import (
	"errors"
	"net/http"
	"strings"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) fail(c *gin.Context, op string, err error) {
	switch {
	case errors.Is(err, ErrSelfAction), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrDetailsTooLong):
		httpx.Fail(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrUserNotFound):
		httpx.Fail(c, http.StatusNotFound, err.Error())
	default:
		logger.LogHandlerError(c, op, http.StatusInternalServerError, err)
		httpx.Fail(c, http.StatusInternalServerError, "could not process request")
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil || !httpx.IsUUID(req.UserID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.Block(c.Request.Context(), httpx.UserID(c), httpx.OrganizationID(c), strings.ToLower(req.UserID)); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	target := c.Param("userId")
	if !httpx.IsUUID(target) {
		httpx.Fail(c, http.StatusBadRequest, "invalid user id")
		return
	}
	if err := h.service.Unblock(c.Request.Context(), httpx.UserID(c), strings.ToLower(target)); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	blocks, err := h.service.ListBlocks(c.Request.Context(), httpx.UserID(c), httpx.Limit(c.Query("limit"), 50, 100), httpx.Offset(c.Query("offset"), 10000))
	if err != nil {
		h.fail(c, "safety.list_blocks", err)
		return
	}
	c.JSON(http.StatusOK, BlocksResponse{Blocks: blocks})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil || !httpx.IsUUID(req.UserID) {
		httpx.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}
	if err := h.service.Report(c.Request.Context(), httpx.UserID(c), httpx.OrganizationID(c), strings.ToLower(req.UserID), req.Reason, req.Details); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.JSON(http.StatusAccepted, ReportResponse{Status: "received", Blocked: true})
}
