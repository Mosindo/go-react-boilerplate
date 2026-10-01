package photos

import "github.com/gin-gonic/gin"

// uploadLimit rate-limits mutations; pass a no-op when limits are disabled.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, uploadLimit gin.HandlerFunc) {
	r.GET("/me/photos", requireUser, h.List)
	r.POST("/me/photos", requireUser, uploadLimit, h.Add)
	r.PUT("/me/photos/order", requireUser, h.Reorder)
	r.PUT("/me/photos/:photoId", requireUser, uploadLimit, h.Replace)
	r.DELETE("/me/photos/:photoId", requireUser, h.Delete)
	r.GET("/photos/:photoId/file", requireUser, h.File)
}
