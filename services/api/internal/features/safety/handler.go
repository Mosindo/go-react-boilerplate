package safety

import (
	"errors"
	"net/http"

	platformerrors "example.com/api/internal/platform/errors"
	"example.com/api/internal/platform/logger"
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
	case errors.Is(err, ErrUserNotFound):
		platformerrors.Respond(c, http.StatusNotFound, "user_not_found", "user not found")
	case errors.Is(err, ErrSelfAction), errors.Is(err, ErrInvalidReason), errors.Is(err, ErrInvalidInput):
		platformerrors.Respond(c, http.StatusBadRequest, "invalid_request", err.Error())
	default:
		logger.LogHandlerError(c, operation, http.StatusInternalServerError, err)
		platformerrors.Respond(c, http.StatusInternalServerError, "internal", "internal error")
	}
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, "safety.block.bind", ErrInvalidInput)
		return
	}
	if err := h.service.Block(c.Request.Context(), c.GetString("userID"), req.UserID); err != nil {
		h.fail(c, "safety.block", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	if err := h.service.Unblock(c.Request.Context(), c.GetString("userID"), c.Param("userId")); err != nil {
		h.fail(c, "safety.unblock", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocks(c *gin.Context) {
	resp, err := h.service.ListBlocks(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		h.fail(c, "safety.blocks", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.fail(c, "safety.report.bind", ErrInvalidInput)
		return
	}
	if err := h.service.Report(c.Request.Context(), c.GetString("userID"), req); err != nil {
		h.fail(c, "safety.report", err)
		return
	}
	c.Status(http.StatusNoContent)
}
