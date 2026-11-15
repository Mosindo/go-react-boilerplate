package notifications

import "github.com/gin-gonic/gin"

// Notifications are produced by the server only (matches, messages); there is
// intentionally no endpoint to create one.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/me/badges", requireUser, handler.Badges)
	r.GET("/notifications", requireUser, handler.List)
	r.POST("/notifications/read-all", requireUser, handler.MarkAllRead)
	r.POST("/notifications/:notificationId/read", requireUser, handler.MarkRead)
}
