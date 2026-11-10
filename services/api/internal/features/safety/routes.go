package safety

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, reportLimit gin.HandlerFunc) {
	g := r.Group("", requireUser)
	g.GET("/blocks", h.ListBlocked)
	g.POST("/blocks", h.Block)
	g.DELETE("/blocks/:userId", h.Unblock)
	g.POST("/reports", reportLimit, h.Report)
}
