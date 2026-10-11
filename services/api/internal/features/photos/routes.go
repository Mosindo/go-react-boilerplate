package photos

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, uploadsPerHour int) {
	upload := middleware.RateLimit(uploadsPerHour, time.Hour, middleware.ByUser)
	r.POST("/me/photos", requireUser, upload, h.Upload)
	r.PUT("/me/photos/order", requireUser, h.Reorder)
	r.PUT("/me/photos/:id", requireUser, upload, h.Replace)
	r.DELETE("/me/photos/:id", requireUser, h.Delete)
	r.GET("/photos/:id/content", h.Content)
}
