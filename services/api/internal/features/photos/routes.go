package photos

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts photo routes. `uploadLimit` throttles writes.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, uploadLimit gin.HandlerFunc) {
	r.GET("/photos/:id/file", h.File)

	g := r.Group("/photos", requireUser)
	g.GET("", h.List)
	g.POST("", uploadLimit, h.Upload)
	g.PUT("/order", h.Reorder)
	g.PUT("/:id", uploadLimit, h.Replace)
	g.PUT("/:id/primary", h.MakePrimary)
	g.DELETE("/:id", h.Delete)
}
