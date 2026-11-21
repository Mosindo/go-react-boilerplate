package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, messageLimit gin.HandlerFunc) {
	r.GET("/chats", requireUser, handler.Chats)
	r.GET("/chats/:userId/messages", requireUser, handler.ChatMessages)
	r.POST("/chats/:userId/messages", requireUser, messageLimit, handler.SendChatMessage)
	r.DELETE("/chats/:userId/messages", requireUser, handler.Clear)
	r.POST("/chats/:userId/read", requireUser, handler.MarkRead)
}
