package alerts

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/notifications", requireUser, h.List)
	r.POST("/notifications/read-all", requireUser, h.MarkAllRead)
	r.POST("/notifications/:notificationId/read", requireUser, h.MarkRead)
}
