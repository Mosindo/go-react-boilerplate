package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, messageLimit gin.HandlerFunc) {
	r.GET("/conversations", requireUser, h.List)
	r.GET("/conversations/unread-count", requireUser, h.Unread)
	r.GET("/conversations/:conversationId", requireUser, h.Get)
	r.DELETE("/conversations/:conversationId", requireUser, h.Clear)
	r.GET("/conversations/:conversationId/messages", requireUser, h.Messages)
	r.POST("/conversations/:conversationId/messages", requireUser, messageLimit, h.Send)
	r.POST("/conversations/:conversationId/read", requireUser, h.MarkRead)
}
