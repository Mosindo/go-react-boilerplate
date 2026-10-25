package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, sockets *SocketHandler, requireUser, messageLimit gin.HandlerFunc) {
	r.GET("/conversations", requireUser, h.ListConversations)
	r.GET("/conversations/unread-count", requireUser, h.UnreadCount)
	r.GET("/conversations/:id/messages", requireUser, h.ListMessages)
	r.POST("/conversations/:id/messages", requireUser, messageLimit, h.Send)
	r.POST("/conversations/:id/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:id", requireUser, h.Hide)
	r.GET("/ws", sockets.Serve)
}
