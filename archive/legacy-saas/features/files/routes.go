package files

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	r.GET("/files", requireUser, handler.List)
	r.POST("/files", requireUser, handler.Create)
	r.GET("/files/:fileId", requireUser, handler.GetByID)
}
