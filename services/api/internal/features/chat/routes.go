package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, messageLimit gin.HandlerFunc) {
	r.GET("/conversations/:conversationId/messages", requireUser, h.ListMessages)
	r.POST("/conversations/:conversationId/messages", requireUser, messageLimit, h.Send)
	r.POST("/conversations/:conversationId/read", requireUser, h.MarkRead)
}
