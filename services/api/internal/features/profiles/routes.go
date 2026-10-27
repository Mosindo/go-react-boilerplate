package profiles

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, uploads *middleware.RateLimiter) {
	r.GET("/profile/me", requireUser, h.GetMine)
	r.PUT("/profile/me", requireUser, h.PutMine)
	r.PUT("/profile/preferences", requireUser, h.PutPreferences)
	r.PUT("/profile/privacy", requireUser, h.PutPrivacy)
	r.GET("/interests", requireUser, h.ListInterests)

	upload := middleware.RateLimit(uploads)
	r.POST("/photos", requireUser, upload, h.AddPhoto)
	r.PUT("/photos/order", requireUser, h.ReorderPhotos)
	r.PUT("/photos/:id", requireUser, upload, h.ReplacePhoto)
	r.DELETE("/photos/:id", requireUser, h.DeletePhoto)
	r.GET("/photos/:id", requireUser, h.ServePhoto)
}
