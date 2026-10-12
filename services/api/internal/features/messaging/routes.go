package messaging

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, sendLimit gin.HandlerFunc) {
	r.GET("/matches/:matchId/messages", requireUser, h.List)
	r.POST("/matches/:matchId/messages", requireUser, sendLimit, h.Send)
	r.POST("/matches/:matchId/read", requireUser, h.MarkRead)
	r.DELETE("/matches/:matchId/messages", requireUser, h.Clear)
}
