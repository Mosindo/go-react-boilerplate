package photos

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts photo routes. uploadLimit throttles uploads per user;
// middlewares (auth first) apply to every route.
func RegisterRoutes(r gin.IRouter, h *Handler, uploadLimit gin.HandlerFunc, middlewares ...gin.HandlerFunc) {
	g := r.Group("", middlewares...)
	g.GET("/me/photos", h.List)
	g.POST("/me/photos", uploadLimit, h.Upload)
	g.PUT("/me/photos/order", h.Reorder)
	g.PUT("/me/photos/:id", uploadLimit, h.Replace)
	g.DELETE("/me/photos/:id", h.Delete)
	g.GET("/photos/:id", h.Serve)
}
