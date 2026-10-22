package photos

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, uploadLimiter *middleware.RateLimiter) {
	upload := uploadLimiter.ByUser()
	r.GET("/photos", requireUser, h.List)
	r.POST("/photos", requireUser, upload, h.Upload)
	r.PUT("/photos/order", requireUser, h.Reorder)
	r.PUT("/photos/:id", requireUser, upload, h.Replace)
	r.PUT("/photos/:id/primary", requireUser, h.SetPrimary)
	r.DELETE("/photos/:id", requireUser, h.Delete)
	r.GET("/photos/:id/file", requireUser, h.File)
}
