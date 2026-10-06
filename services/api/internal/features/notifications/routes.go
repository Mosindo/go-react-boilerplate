package notifications

import "github.com/gin-gonic/gin"

// Notifications are created by the server (matches, messages); clients can
// only read them and mark them as read.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/notifications", requireUser, h.List)
	r.POST("/notifications/read-all", requireUser, h.MarkAllRead)
	r.POST("/notifications/:notificationId/read", requireUser, h.MarkRead)
}
