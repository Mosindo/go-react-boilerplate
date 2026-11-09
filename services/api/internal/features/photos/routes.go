package photos

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, uploadLimit gin.HandlerFunc) {
	r.GET("/me/photos", requireUser, h.List)
	r.POST("/me/photos", requireUser, uploadLimit, h.Upload)
	r.PUT("/me/photos/order", requireUser, h.Reorder)
	r.PUT("/me/photos/:id", requireUser, uploadLimit, h.Replace)
	r.PUT("/me/photos/:id/primary", requireUser, h.MakePrimary)
	r.DELETE("/me/photos/:id", requireUser, h.Delete)
	r.GET("/photos/:id", requireUser, h.Serve)
}
