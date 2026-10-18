package chat

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, sends *middleware.RateLimiter) {
	r.GET("/conversations", requireUser, h.Conversations)
	r.GET("/conversations/:conversationId/messages", requireUser, h.Messages)
	r.POST("/conversations/:conversationId/messages", requireUser, sends.ByUser("message"), h.Send)
	r.POST("/conversations/:conversationId/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:conversationId", requireUser, h.Hide)
}
