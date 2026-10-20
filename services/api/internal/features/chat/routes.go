package chat

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, sendLimiter *middleware.Limiter) {
	r.GET("/conversations", requireUser, handler.ListConversations)
	r.GET("/conversations/:id/messages", requireUser, handler.ListMessages)
	r.POST("/conversations/:id/messages", requireUser, middleware.RateLimitByUser(sendLimiter), handler.Send)
	r.POST("/conversations/:id/read", requireUser, handler.MarkRead)
	r.DELETE("/conversations/:id", requireUser, handler.Hide)
}
