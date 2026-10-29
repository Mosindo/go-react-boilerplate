package safety

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, reportLimit gin.HandlerFunc) {
	if reportLimit == nil {
		reportLimit = func(c *gin.Context) { c.Next() }
	}
	r.GET("/blocks", requireUser, handler.ListBlocked)
	r.POST("/blocks", requireUser, handler.Block)
	r.DELETE("/blocks/:userId", requireUser, handler.Unblock)
	r.POST("/reports", requireUser, reportLimit, handler.Report)
}
