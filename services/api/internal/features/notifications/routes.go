package notifications

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, middlewares ...gin.HandlerFunc) {
	g := r.Group("", middlewares...)
	g.GET("/notifications", h.List)
	g.POST("/notifications/read-all", h.MarkAllRead)
	g.POST("/notifications/:id/read", h.MarkRead)
}
