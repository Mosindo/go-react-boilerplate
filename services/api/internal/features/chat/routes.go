package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, sendLimit gin.HandlerFunc) {
	r.GET("/conversations", requireUser, h.ListConversations)
	r.GET("/conversations/:id/messages", requireUser, h.ListMessages)
	r.POST("/conversations/:id/messages", requireUser, sendLimit, h.Send)
	r.POST("/conversations/:id/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:id", requireUser, h.Clear)
}
