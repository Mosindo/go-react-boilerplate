package photos

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, uploadLimit *middleware.RateLimiter) {
	r.POST("/profile/photos", requireUser, uploadLimit.ByUser(), h.Upload)
	r.PUT("/profile/photos/order", requireUser, h.Reorder)
	r.DELETE("/profile/photos/:id", requireUser, h.Delete)
	r.GET("/photos/:id/file", requireUser, h.File)
}
