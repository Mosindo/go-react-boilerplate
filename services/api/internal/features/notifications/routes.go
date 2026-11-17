package notifications

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/notifications", requireUser, handler.List)
	r.GET("/notifications/unread-count", requireUser, handler.UnreadCount)
	r.POST("/notifications/read-all", requireUser, handler.MarkAllRead)
	r.POST("/notifications/:notificationId/read", requireUser, handler.MarkRead)
}
