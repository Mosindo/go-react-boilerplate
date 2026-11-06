package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, sendLimit gin.HandlerFunc) {
	g := r.Group("/conversations", requireUser)
	g.GET("", h.Conversations)
	g.GET("/:conversationId/messages", h.Messages)
	g.POST("/:conversationId/messages", sendLimit, h.Send)
	g.POST("/:conversationId/read", h.MarkRead)
	g.DELETE("/:conversationId", h.Hide)
}
