package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/chats", requireUser, handler.Chats)
	r.GET("/chats/:userId/messages", requireUser, handler.ChatMessages)
	r.POST("/chats/:userId/messages", requireUser, handler.SendChatMessage)
}
