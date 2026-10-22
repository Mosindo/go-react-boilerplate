package chat

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, messageLimiter *middleware.RateLimiter) {
	r.GET("/conversations", requireUser, h.ListConversations)
	r.GET("/conversations/:id/messages", requireUser, h.ListMessages)
	r.POST("/conversations/:id/messages", requireUser, messageLimiter.ByUser(), h.Send)
	r.POST("/conversations/:id/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:id", requireUser, h.Hide)
}
