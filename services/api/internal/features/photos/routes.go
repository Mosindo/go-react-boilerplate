package photos

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, uploadLimit gin.HandlerFunc) {
	r.GET("/me/photos", requireUser, handler.List)
	r.POST("/me/photos", requireUser, uploadLimit, handler.Upload)
	r.POST("/me/photos/reorder", requireUser, handler.Reorder)
	r.PUT("/me/photos/:photoId", requireUser, uploadLimit, handler.Replace)
	r.DELETE("/me/photos/:photoId", requireUser, handler.Delete)
	r.GET("/media/photos/:photoId", handler.Serve)
}
