package chat

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	sendLimit := middleware.RateLimitByUser(middleware.NewLimiter(60, time.Minute))
	r.GET("/conversations", requireUser, h.Conversations)
	r.GET("/conversations/unread", requireUser, h.Unread)
	r.GET("/conversations/:id/messages", requireUser, h.Messages)
	r.POST("/conversations/:id/messages", requireUser, sendLimit, h.Send)
	r.POST("/conversations/:id/read", requireUser, h.Read)
	r.DELETE("/conversations/:id", requireUser, h.Hide)
}
