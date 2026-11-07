package chat

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, ws *WSHandler, requireUser gin.HandlerFunc, sendLimiter *middleware.Limiter) {
	r.GET("/ws", ws.Serve)
	r.GET("/conversations", requireUser, h.List)
	r.GET("/conversations/:id/messages", requireUser, h.Messages)
	r.POST("/conversations/:id/messages", requireUser, middleware.RateLimitByUser(sendLimiter), h.Send)
	r.POST("/conversations/:id/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:id", requireUser, h.Clear)
}
