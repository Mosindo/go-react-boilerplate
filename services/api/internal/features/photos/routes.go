package photos

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts the photo routes. uploadLimit throttles mutations per user.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser, uploadLimit gin.HandlerFunc) {
	upload := middleware.MaxBody(MaxUploadBytes + 1<<20)
	r.POST("/photos", requireUser, uploadLimit, upload, h.Add)
	r.PUT("/photos/order", requireUser, h.Reorder)
	r.PUT("/photos/:id", requireUser, uploadLimit, upload, h.Replace)
	r.DELETE("/photos/:id", requireUser, h.Delete)
	r.GET("/photos/:id/image", requireUser, h.Image)
}
