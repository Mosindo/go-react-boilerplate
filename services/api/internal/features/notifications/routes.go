package notifications

import "github.com/gin-gonic/gin"

// Notifications are created by the server only (matches, messages); clients
// can read them and mark them as read.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/notifications", requireUser, handler.List)
	r.GET("/notifications/unread-count", requireUser, handler.UnreadCount)
	r.POST("/notifications/read-all", requireUser, handler.MarkAllRead)
	r.POST("/notifications/:notificationId/read", requireUser, handler.MarkRead)
}
