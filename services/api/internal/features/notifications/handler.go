package notifications

import (
	"net/http"
	"strconv"

	"example.com/api/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	resp, err := h.service.List(c.Request.Context(), httpx.UserID(c), httpx.QueryLimit(c, 20, 50), offset)
	if err != nil {
		httpx.Fail(c, "notifications.list", err)
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := httpx.UUIDParam(c, "notificationId")
	if !ok {
		return
	}
	n, err := h.service.MarkRead(c.Request.Context(), httpx.UserID(c), id)
	if err != nil {
		httpx.Fail(c, "notifications.mark_read", err)
		return
	}
	c.JSON(http.StatusOK, n)
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	if err := h.service.MarkAllRead(c.Request.Context(), httpx.UserID(c)); err != nil {
		httpx.Fail(c, "notifications.mark_all_read", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) RegisterPushToken(c *gin.Context) {
	var req PushTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "notifications.push_token.bind", err)
		return
	}
	if err := h.service.RegisterPushToken(c.Request.Context(), httpx.UserID(c), req.Token, req.Platform); err != nil {
		httpx.Fail(c, "notifications.push_token", err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) UnregisterPushToken(c *gin.Context) {
	var req DeletePushTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BadRequest(c, "notifications.push_token_delete.bind", err)
		return
	}
	if err := h.service.UnregisterPushToken(c.Request.Context(), httpx.UserID(c), req.Token); err != nil {
		httpx.Fail(c, "notifications.push_token_delete", err)
		return
	}
	c.Status(http.StatusNoContent)
}
