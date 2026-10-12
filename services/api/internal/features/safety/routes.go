package safety

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, reportLimit gin.HandlerFunc) {
	r.GET("/blocks", requireUser, h.List)
	r.POST("/blocks", requireUser, h.Block)
	r.DELETE("/blocks/:userId", requireUser, h.Unblock)
	r.POST("/reports", requireUser, reportLimit, h.Report)
}
