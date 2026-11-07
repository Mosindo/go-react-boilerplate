package profiles

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, uploadLimiter *middleware.Limiter) {
	r.GET("/media/:id", h.Media)
	r.GET("/interests", requireUser, h.Interests)
	r.GET("/profile/me", requireUser, h.GetMine)
	r.PUT("/profile/me", requireUser, middleware.RateLimitByUser(uploadLimiter), h.Upsert)
	r.POST("/profile/photos", requireUser, middleware.RateLimitByUser(uploadLimiter), h.UploadPhoto)
	r.PUT("/profile/photos/order", requireUser, h.ReorderPhotos)
	r.DELETE("/profile/photos/:id", requireUser, h.DeletePhoto)
	r.GET("/profiles/:id", requireUser, h.View)
}
