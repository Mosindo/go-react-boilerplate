package photos

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, uploadLimiter *middleware.RateLimiter) {
	r.GET("/photos/:id", requireUser, h.Get)
	r.GET("/me/photos", requireUser, h.List)
	r.POST("/me/photos", requireUser, uploadLimiter.ByUser(), h.Upload)
	r.PUT("/me/photos/order", requireUser, h.Reorder)
	r.PUT("/me/photos/:id", requireUser, uploadLimiter.ByUser(), h.Replace)
	r.DELETE("/me/photos/:id", requireUser, h.Delete)
}
