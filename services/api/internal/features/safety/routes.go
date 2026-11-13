package safety

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, reportLimit gin.HandlerFunc) {
	r.GET("/blocks", requireUser, handler.ListBlocked)
	r.POST("/blocks", requireUser, handler.Block)
	r.DELETE("/blocks/:userId", requireUser, handler.Unblock)
	r.POST("/reports", requireUser, reportLimit, handler.Report)
}
