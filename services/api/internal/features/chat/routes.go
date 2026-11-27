package chat

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, sendLimit gin.HandlerFunc) {
	r.GET("/matches/:matchId/messages", requireUser, handler.ListMessages)
	r.POST("/matches/:matchId/messages", requireUser, sendLimit, handler.Send)
	r.POST("/matches/:matchId/read", requireUser, handler.MarkRead)
}
