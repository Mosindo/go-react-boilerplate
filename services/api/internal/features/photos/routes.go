package photos

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, uploadLimit gin.HandlerFunc) {
	me := r.Group("/me/photos", requireUser)
	me.GET("", h.List)
	me.POST("", uploadLimit, h.Upload)
	me.PUT("/order", h.Reorder)
	me.PUT("/:photoId", uploadLimit, h.Replace)
	me.DELETE("/:photoId", h.Delete)
	r.GET("/photos/:photoId", requireUser, h.Serve)
}
