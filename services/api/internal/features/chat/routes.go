package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, sendLimit gin.HandlerFunc) {
	r.GET("/conversations", requireUser, h.List)
	r.GET("/conversations/:conversationId/messages", requireUser, h.Messages)
	r.POST("/conversations/:conversationId/messages", requireUser, sendLimit, h.Send)
	r.POST("/conversations/:conversationId/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:conversationId", requireUser, h.Clear)
}
