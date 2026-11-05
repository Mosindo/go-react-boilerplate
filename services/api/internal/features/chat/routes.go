package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	g := r.Group("/conversations", requireUser)
	g.GET("", h.List)
	g.GET("/:id/messages", h.Messages)
	g.POST("/:id/messages", h.Send)
	g.POST("/:id/read", h.MarkRead)
	g.DELETE("/:id", h.Hide)
}
