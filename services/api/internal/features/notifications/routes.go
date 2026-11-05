package notifications

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	g := r.Group("/notifications", requireUser)
	g.GET("", h.List)
	g.GET("/unread-count", h.UnreadCount)
	g.POST("/read-all", h.MarkAllRead)
	g.POST("/:id/read", h.MarkRead)
}
