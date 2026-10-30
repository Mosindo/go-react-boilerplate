package chat

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/conversations", requireUser, h.ListConversations)
	r.GET("/conversations/:id/messages", requireUser, h.ListMessages)
	r.POST("/conversations/:id/messages", requireUser, middleware.RateLimit(60, time.Minute), h.Send)
	r.POST("/conversations/:id/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:id", requireUser, h.Hide)
	r.GET("/events", requireUser, h.Events)
}
