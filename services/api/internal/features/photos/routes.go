package photos

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	upload := middleware.RateLimitByUser(20, 10*time.Minute)

	r.GET("/photos", requireUser, h.List)
	r.POST("/photos", requireUser, upload, h.Upload)
	r.PUT("/photos/order", requireUser, h.Reorder)
	r.GET("/photos/:photoId/content", requireUser, h.Content)
	r.DELETE("/photos/:photoId", requireUser, h.Delete)
}
