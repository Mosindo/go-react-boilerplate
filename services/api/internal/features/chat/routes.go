package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, messageLimit gin.HandlerFunc) {
	r.GET("/matches/:matchId/messages", requireUser, h.List)
	r.POST("/matches/:matchId/messages", requireUser, messageLimit, h.Send)
	r.POST("/matches/:matchId/read", requireUser, h.MarkRead)
}
