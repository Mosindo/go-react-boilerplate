package chat

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, ws *WSHandler, requireUser gin.HandlerFunc, sendLimiter *middleware.RateLimiter) {
	r.GET("/ws", ws.Serve)
	r.GET("/conversations", requireUser, h.Conversations)
	r.GET("/conversations/:id/messages", requireUser, h.Messages)
	r.POST("/conversations/:id/messages", requireUser, sendLimiter.ByUser(), h.Send)
	r.POST("/conversations/:id/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:id", requireUser, h.Hide)
}
