package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, ws *WSHandler, requireUser, sendLimit gin.HandlerFunc) {
	r.GET("/ws", ws.Serve)

	g := r.Group("/conversations", requireUser)
	g.GET("", h.List)
	g.GET("/:id/messages", h.Messages)
	g.POST("/:id/messages", sendLimit, h.Send)
	g.POST("/:id/read", h.MarkRead)
	g.POST("/:id/unmatch", h.Unmatch)
	g.DELETE("/:id", h.Hide)
}
