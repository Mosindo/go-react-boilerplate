package photos

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, uploadLimit gin.HandlerFunc) {
	r.GET("/profile/photos", requireUser, h.List)
	r.POST("/profile/photos", requireUser, uploadLimit, h.Upload)
	r.PUT("/profile/photos/order", requireUser, h.Reorder)
	r.PUT("/profile/photos/:photoId", requireUser, uploadLimit, h.Replace)
	r.DELETE("/profile/photos/:photoId", requireUser, h.Delete)
}
