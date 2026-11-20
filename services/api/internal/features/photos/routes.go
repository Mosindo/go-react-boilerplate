package photos

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, uploadLimit gin.HandlerFunc) {
	r.GET("/profile/photos", requireUser, handler.List)
	r.POST("/profile/photos", requireUser, uploadLimit, handler.Upload)
	r.PUT("/profile/photos/order", requireUser, handler.Reorder)
	r.PUT("/profile/photos/:photoId", requireUser, uploadLimit, handler.Replace)
	r.DELETE("/profile/photos/:photoId", requireUser, handler.Delete)
	r.GET("/media/photos/:photoId", handler.Media)
}
