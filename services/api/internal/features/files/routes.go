package files

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts photo management (authenticated) and the signed file
// endpoint (authorized by its signature). uploadLimit throttles writes per user.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser, uploadLimit gin.HandlerFunc) {
	r.GET("/photos", requireUser, handler.List)
	r.POST("/photos", requireUser, uploadLimit, handler.Upload)
	r.PUT("/photos/order", requireUser, handler.Reorder)
	r.PUT("/photos/:photoId", requireUser, uploadLimit, handler.Replace)
	r.POST("/photos/:photoId/main", requireUser, handler.SetMain)
	r.DELETE("/photos/:photoId", requireUser, handler.Delete)

	r.GET("/photos/:photoId/file", handler.File)
}
