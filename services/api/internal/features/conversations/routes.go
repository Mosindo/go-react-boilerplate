package conversations

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, sendLimiter gin.HandlerFunc) {
	r.GET("/conversations", requireUser, h.List)
	r.GET("/conversations/:matchId/messages", requireUser, h.Messages)
	r.POST("/conversations/:matchId/messages", requireUser, sendLimiter, h.Send)
	r.POST("/conversations/:matchId/read", requireUser, h.MarkRead)
	r.DELETE("/conversations/:matchId", requireUser, h.Hide)
	r.DELETE("/matches/:matchId", requireUser, h.Unmatch)
}
