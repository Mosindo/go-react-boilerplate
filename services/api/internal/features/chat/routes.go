package chat

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, messageLimiter *middleware.RateLimiter) {
	r.GET("/chats", requireUser, h.Chats)
	r.GET("/chats/:userId/messages", requireUser, h.Messages)
	r.POST("/chats/:userId/messages", requireUser, middleware.RateLimitByUser(messageLimiter, "messages"), h.Send)
	r.POST("/chats/:userId/read", requireUser, h.MarkRead)
	r.DELETE("/chats/:userId", requireUser, h.Clear)
}
