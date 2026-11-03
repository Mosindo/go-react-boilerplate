package photos

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, uploadLimit gin.HandlerFunc) {
	r.POST("/me/photos", requireUser, uploadLimit, h.Upload)
	r.PUT("/me/photos/order", requireUser, h.Reorder)
	r.POST("/me/photos/:id/replace", requireUser, uploadLimit, h.Replace)
	r.DELETE("/me/photos/:id", requireUser, h.Delete)
	r.GET("/photos/:id", requireUser, h.Serve)
}
