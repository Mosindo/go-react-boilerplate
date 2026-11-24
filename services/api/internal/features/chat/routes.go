package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, sendRL gin.HandlerFunc) {
	r.GET("/conversations", requireUser, h.ListConversations)
	r.GET("/conversations/:conversationId/messages", requireUser, h.ListMessages)
	r.POST("/conversations/:conversationId/messages", requireUser, sendRL, h.Send)
	r.POST("/conversations/:conversationId/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:conversationId", requireUser, h.Hide)
}
