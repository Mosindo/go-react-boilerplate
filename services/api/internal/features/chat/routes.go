package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, messageLimit gin.HandlerFunc) {
	r.GET("/chats", requireUser, h.Chats)
	r.GET("/chats/unread", requireUser, h.Unread)
	r.GET("/chats/:userId/messages", requireUser, h.Messages)
	r.POST("/chats/:userId/messages", requireUser, messageLimit, h.Send)
	r.POST("/chats/:userId/read", requireUser, h.MarkRead)
	r.DELETE("/chats/:userId", requireUser, h.Hide)
}
