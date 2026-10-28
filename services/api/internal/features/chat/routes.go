package chat

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	send := middleware.RateLimitByUser(middleware.NewLimiter(60, time.Minute))
	read := middleware.RateLimitByUser(middleware.NewLimiter(600, time.Minute))

	r.GET("/conversations", requireUser, read, h.List)
	r.GET("/conversations/:id/messages", requireUser, read, h.Messages)
	r.POST("/conversations/:id/messages", requireUser, send, h.Send)
	r.POST("/conversations/:id/read", requireUser, read, h.Read)
	r.DELETE("/conversations/:id", requireUser, h.Clear)
	r.POST("/conversations/:id/unmatch", requireUser, h.Unmatch)
}
