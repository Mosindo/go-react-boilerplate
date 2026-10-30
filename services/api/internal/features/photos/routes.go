package photos

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/photos", requireUser, h.List)
	r.POST("/photos", requireUser, middleware.RateLimit(20, 10*time.Minute), h.Upload)
	r.PUT("/photos/order", requireUser, h.Reorder)
	r.PUT("/photos/:id", requireUser, middleware.RateLimit(20, 10*time.Minute), h.Replace)
	r.DELETE("/photos/:id", requireUser, h.Delete)
	// Authorized by the signed URL itself, so it can be loaded by an <Image>.
	r.GET("/photos/:id/file", h.File)
}
