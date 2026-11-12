package notifications

import "github.com/gin-gonic/gin"

// Notifications are produced by the server only (matches, messages);
// clients can read them and mark them as read.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/notifications", requireUser, h.List)
	r.GET("/notifications/unread-count", requireUser, h.UnreadCount)
	r.POST("/notifications/read-all", requireUser, h.MarkAllRead)
	r.POST("/notifications/:notificationId/read", requireUser, h.MarkRead)
}
