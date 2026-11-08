package photos

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, uploads *middleware.Limiter) {
	r.GET("/photos", requireUser, h.List)
	r.POST("/photos", requireUser, uploads.ByUser(), h.Upload)
	r.PUT("/photos/order", requireUser, h.Reorder)
	r.PUT("/photos/:photoId", requireUser, uploads.ByUser(), h.Replace)
	r.DELETE("/photos/:photoId", requireUser, h.Delete)
	r.GET("/photos/:photoId/file", requireUser, h.File)
}
