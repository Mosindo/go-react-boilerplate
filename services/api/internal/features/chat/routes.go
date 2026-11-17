package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, sendLimit gin.HandlerFunc) {
	r.GET("/conversations", requireUser, handler.ListConversations)
	r.GET("/conversations/unread-count", requireUser, handler.UnreadCount)
	r.GET("/conversations/:conversationId", requireUser, handler.GetConversation)
	r.DELETE("/conversations/:conversationId", requireUser, handler.Hide)
	r.GET("/conversations/:conversationId/messages", requireUser, handler.ListMessages)
	r.POST("/conversations/:conversationId/messages", requireUser, sendLimit, handler.SendMessage)
	r.POST("/conversations/:conversationId/read", requireUser, handler.MarkRead)
}
