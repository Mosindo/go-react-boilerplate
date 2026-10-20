package photos

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, uploadLimiter *middleware.Limiter) {
	r.POST("/photos", requireUser, middleware.RateLimitByUser(uploadLimiter), handler.Upload)
	r.PUT("/photos/order", requireUser, handler.Reorder)
	r.DELETE("/photos/:id", requireUser, handler.Delete)
	r.GET("/photos/:id/file", requireUser, handler.File)
}
