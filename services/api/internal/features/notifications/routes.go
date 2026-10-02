package notifications

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/notifications", requireUser, h.List)
	r.GET("/notifications/summary", requireUser, h.Summary)
	r.POST("/notifications/read", requireUser, h.MarkAllRead)
	r.POST("/notifications/:id/read", requireUser, h.MarkRead)
}
