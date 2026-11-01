package notifications

import (
	"net/http"

	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) List(c *gin.Context) {
	limit, ok := httpx.Limit(c, 30, 50)
	cur, okCur := httpx.DecodeCursor(c)
	if !ok || !okCur {
		httpx.Error(c, http.StatusBadRequest, "invalid pagination")
		return
	}
	items, unread, err := h.service.List(c.Request.Context(), c.GetString("userID"), limit, cur)
	if err != nil {
		logger.LogHandlerError(c, "notifications.list", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	next := ""
	if len(items) == limit {
		last := items[len(items)-1]
		next = httpx.EncodeCursor(last.CreatedAt, last.ID)
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"items": items, "unreadCount": unread, "nextCursor": next})
}

func (h *Handler) MarkRead(c *gin.Context) {
	var req MarkReadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid request")
		return
	}
	for _, id := range req.IDs {
		if !httpx.IsUUID(id) {
			httpx.Error(c, http.StatusBadRequest, "invalid request")
			return
		}
	}
	if err := h.service.MarkRead(c.Request.Context(), c.GetString("userID"), req.IDs); err != nil {
		logger.LogHandlerError(c, "notifications.read", http.StatusInternalServerError, err)
		httpx.Internal(c)
		return
	}
	c.Status(http.StatusNoContent)
}
