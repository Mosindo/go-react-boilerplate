package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, messageLimit gin.HandlerFunc) {
	r.GET("/conversations", requireUser, handler.List)
	r.GET("/conversations/:conversationId", requireUser, handler.Get)
	r.DELETE("/conversations/:conversationId", requireUser, handler.Clear)
	r.GET("/conversations/:conversationId/messages", requireUser, handler.Messages)
	r.POST("/conversations/:conversationId/messages", requireUser, messageLimit, handler.Send)
	r.POST("/conversations/:conversationId/read", requireUser, handler.MarkRead)
}
