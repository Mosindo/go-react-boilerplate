package photos

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, limiter *middleware.RateLimiter) {
	upload := limiter.ByUser("photo-upload", 30, time.Hour)
	r.GET("/photos", requireUser, h.List)
	r.POST("/photos", requireUser, upload, h.Upload)
	r.PUT("/photos/order", requireUser, h.Reorder)
	r.PUT("/photos/:id", requireUser, upload, h.Replace)
	r.DELETE("/photos/:id", requireUser, h.Delete)
	r.GET("/photos/:id/file", requireUser, limiter.ByUser("photo-read", 1200, time.Minute), h.File)
}
