package chat

import (
	"time"

	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	send := ratelimit.ByUser("send-message", ratelimit.New(60, time.Minute))

	r.GET("/conversations", requireUser, h.ListConversations)
	r.GET("/conversations/:id/messages", requireUser, h.ListMessages)
	r.POST("/conversations/:id/messages", requireUser, send, h.Send)
	r.POST("/conversations/:id/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:id", requireUser, h.Clear)
}
