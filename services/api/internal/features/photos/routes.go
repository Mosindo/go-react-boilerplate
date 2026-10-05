package photos

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, uploadLimit gin.HandlerFunc) {
	r.GET("/photos", requireUser, h.List)
	r.POST("/photos", requireUser, uploadLimit, h.Upload)
	r.PUT("/photos/order", requireUser, h.Reorder)
	r.PUT("/photos/:photoId", requireUser, uploadLimit, h.Replace)
	r.DELETE("/photos/:photoId", requireUser, h.Delete)
	r.GET("/photos/:photoId/image", h.Image)
}
