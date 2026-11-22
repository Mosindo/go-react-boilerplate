package photos

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, uploadLimiter *middleware.RateLimiter) {
	// Body size is capped globally by middleware.BodyLimits (multipart branch).
	limit := middleware.RateLimitByUser(uploadLimiter, "photos.upload")

	r.GET("/me/photos", requireUser, h.List)
	r.POST("/me/photos", requireUser, limit, h.Upload)
	r.PUT("/me/photos/order", requireUser, h.Reorder)
	r.PUT("/me/photos/:photoId", requireUser, limit, h.Replace)
	r.DELETE("/me/photos/:photoId", requireUser, h.Delete)
	r.GET("/media/photos/:photoId", h.Serve)
}
