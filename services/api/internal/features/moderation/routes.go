package moderation

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, reportLimit gin.HandlerFunc) {
	r.GET("/blocks", requireUser, h.List)
	r.POST("/users/:id/block", requireUser, h.Block)
	r.DELETE("/users/:id/block", requireUser, h.Unblock)
	r.POST("/users/:id/report", requireUser, reportLimit, h.Report)
}
