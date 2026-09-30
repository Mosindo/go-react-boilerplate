package moderation

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	g := r.Group("/moderation", requireUser, handler.RequireModerator)
	g.GET("/reports", handler.ListReports)
	g.POST("/reports/:reportId/resolve", handler.ResolveReport)
	g.POST("/users/:userId/suspend", handler.Suspend)
	g.POST("/users/:userId/unsuspend", handler.Unsuspend)
}
