package chat

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, messagesPerMinute int) {
	send := middleware.RateLimit(messagesPerMinute, time.Minute, middleware.ByUser)
	r.GET("/conversations", requireUser, h.List)
	r.DELETE("/conversations/:id", requireUser, h.Hide)
	r.GET("/conversations/:id/messages", requireUser, h.Messages)
	r.POST("/conversations/:id/messages", requireUser, send, h.Send)
	r.POST("/conversations/:id/read", requireUser, h.MarkRead)
}
