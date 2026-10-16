package chat

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	readLimit := middleware.NewLimiter(240, time.Minute).ByUser()
	sendLimit := middleware.NewLimiter(60, time.Minute).ByUser()

	r.GET("/conversations", requireUser, readLimit, h.Conversations)
	r.GET("/conversations/unread-count", requireUser, readLimit, h.Unread)
	r.GET("/conversations/:matchId/messages", requireUser, readLimit, h.Messages)
	r.POST("/conversations/:matchId/messages", requireUser, sendLimit, h.Send)
	r.POST("/conversations/:matchId/read", requireUser, readLimit, h.MarkRead)
	r.DELETE("/conversations/:matchId", requireUser, readLimit, h.Clear)
}
