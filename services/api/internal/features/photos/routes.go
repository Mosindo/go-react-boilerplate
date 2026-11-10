package photos

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, uploadLimit gin.HandlerFunc) {
	g := r.Group("/profile/photos", requireUser)
	g.GET("", h.List)
	g.POST("", uploadLimit, h.Upload)
	g.PUT("/order", h.Reorder)
	g.PUT("/:photoId", uploadLimit, h.Replace)
	g.DELETE("/:photoId", h.Delete)

	r.GET("/media/:key", h.Serve)
}
