package moderation

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, middlewares ...gin.HandlerFunc) {
	g := r.Group("", middlewares...)
	g.GET("/blocks", h.ListBlocks)
	g.POST("/blocks", h.Block)
	g.DELETE("/blocks/:userId", h.Unblock)
	g.POST("/reports", h.Report)
}
