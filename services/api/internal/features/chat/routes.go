package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, sendLimit gin.HandlerFunc) {
	g := r.Group("/conversations", requireUser)
	g.GET("", h.ListConversations)
	g.GET("/unread-count", h.UnreadCount)
	g.GET("/:conversationId", h.GetConversation)
	g.GET("/:conversationId/messages", h.ListMessages)
	g.POST("/:conversationId/messages", sendLimit, h.Send)
	g.POST("/:conversationId/read", h.MarkRead)
}
