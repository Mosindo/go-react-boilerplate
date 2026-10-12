package photos

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, uploadLimit gin.HandlerFunc) {
	r.POST("/me/photos", requireUser, uploadLimit, h.Upload)
	r.PUT("/me/photos/order", requireUser, h.Reorder)
	r.PUT("/me/photos/:photoId", requireUser, uploadLimit, h.Replace)
	r.DELETE("/me/photos/:photoId", requireUser, h.Delete)
	r.GET("/media/photos/:photoId", h.Serve)
}
