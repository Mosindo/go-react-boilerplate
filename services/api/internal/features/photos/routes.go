package photos

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/photos/:id", requireUser, h.Content)
	me := r.Group("/me/photos", requireUser)
	me.GET("", h.List)
	me.POST("", h.Upload)
	me.PUT("/order", h.Reorder)
	me.PUT("/:id", h.Replace)
	me.DELETE("/:id", h.Delete)
}
