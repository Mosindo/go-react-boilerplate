package chat

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, sendLimit *middleware.RateLimiter) {
	r.GET("/matches", requireUser, h.ListMatches)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)

	r.GET("/conversations", requireUser, h.ListConversations)
	r.GET("/conversations/unread-count", requireUser, h.UnreadCount)
	r.GET("/conversations/:id/messages", requireUser, h.Messages)
	r.POST("/conversations/:id/messages", requireUser, sendLimit.ByUser(), h.Send)
	r.POST("/conversations/:id/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:id", requireUser, h.Hide)
}
