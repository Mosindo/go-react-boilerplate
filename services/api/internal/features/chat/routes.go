package chat

import "github.com/gin-gonic/gin"

// Conversations are created by the matching module when two people like each
// other; there is no endpoint to start one, so strangers cannot message each other.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, sendLimit gin.HandlerFunc) {
	r.GET("/chats", requireUser, handler.List)
	r.GET("/chats/:conversationId/messages", requireUser, handler.Messages)
	r.POST("/chats/:conversationId/messages", requireUser, sendLimit, handler.Send)
	r.POST("/chats/:conversationId/read", requireUser, handler.MarkRead)
	r.DELETE("/chats/:conversationId", requireUser, handler.Hide)
}
