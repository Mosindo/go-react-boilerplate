package chat

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	send := middleware.RateLimitByUser(60, time.Minute)

	r.GET("/matches", requireUser, h.Matches)
	r.DELETE("/matches/:matchId", requireUser, h.Unmatch)
	r.GET("/matches/:matchId/messages", requireUser, h.Messages)
	r.POST("/matches/:matchId/messages", requireUser, send, h.Send)
	r.POST("/matches/:matchId/read", requireUser, h.MarkRead)
	r.POST("/matches/:matchId/hide", requireUser, h.Hide)
}
