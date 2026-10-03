package photos

import "github.com/gin-gonic/gin"

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, uploadLimiter gin.HandlerFunc) {
	r.GET("/me/photos", requireUser, h.List)
	r.POST("/me/photos", requireUser, uploadLimiter, h.Create)
	r.PUT("/me/photos/order", requireUser, h.Reorder)
	r.PUT("/me/photos/:photoId", requireUser, uploadLimiter, h.Replace)
	r.DELETE("/me/photos/:photoId", requireUser, h.Delete)
	r.GET("/photos/:photoId", requireUser, h.File)
}
