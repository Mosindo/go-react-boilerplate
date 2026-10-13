package chat

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, ws *SocketHandler, requireUser gin.HandlerFunc, limiter *middleware.RateLimiter) {
	r.GET("/conversations", requireUser, h.Conversations)
	r.GET("/conversations/:id/messages", requireUser, h.Messages)
	r.POST("/conversations/:id/messages", requireUser, limiter.ByUser("message", 60, time.Minute), h.Send)
	r.POST("/conversations/:id/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:id", requireUser, h.Hide)
	r.GET("/ws", limiter.ByIP("ws", 60, time.Minute), ws.Serve)
}
