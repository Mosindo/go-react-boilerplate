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

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func writeError(c *gin.Context, op string, err error) {
	var validation *platformerrors.ValidationError
	status := http.StatusInternalServerError
	body := gin.H{"error": "internal error"}
	switch {
	case errors.As(err, &validation):
		status, body = http.StatusBadRequest, gin.H{"error": validation.Message, "field": validation.Field}
	case errors.Is(err, ErrUserNotFound):
		status, body = http.StatusNotFound, gin.H{"error": "user not found"}
	}
	logger.LogHandlerError(c, op, status, err)
	c.JSON(status, body)
}

func (h *Handler) Block(c *gin.Context) {
	var req BlockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.service.Block(c.Request.Context(), c.GetString("userID"), req.UserID); err != nil {
		writeError(c, "blocks.create", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Unblock(c *gin.Context) {
	if err := h.service.Unblock(c.Request.Context(), c.GetString("userID"), c.Param("userId")); err != nil {
		writeError(c, "blocks.delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListBlocked(c *gin.Context) {
	list, err := h.service.ListBlocked(c.Request.Context(), c.GetString("userID"))
	if err != nil {
		writeError(c, "blocks.list", err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"blocks": list})
}

func (h *Handler) Report(c *gin.Context) {
	var req ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if err := h.service.Report(c.Request.Context(), c.GetString("userID"), req); err != nil {
		writeError(c, "reports.create", err)
		return
	}
	c.Status(http.StatusCreated)
}
