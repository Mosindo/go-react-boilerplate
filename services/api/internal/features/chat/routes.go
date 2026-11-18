package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, messageLimit gin.HandlerFunc) {
	r.GET("/chats", requireUser, handler.Chats)
	r.GET("/chats/:userId/messages", requireUser, handler.Messages)
	r.POST("/chats/:userId/messages", requireUser, messageLimit, handler.Send)
	r.POST("/chats/:userId/read", requireUser, handler.MarkRead)
}
