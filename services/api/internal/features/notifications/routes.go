package notifications

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	limit := middleware.NewLimiter(300, time.Minute).ByUser()
	r.GET("/notifications", requireUser, limit, h.List)
	r.GET("/notifications/unread-count", requireUser, limit, h.UnreadCount)
	r.POST("/notifications/read-all", requireUser, h.MarkAllRead)
	r.POST("/notifications/:id/read", requireUser, h.MarkRead)
}
