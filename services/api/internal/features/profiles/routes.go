package profiles

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	authed := r.Group("", requireUser)
	authed.GET("/interests", h.ListInterests)
	authed.GET("/profile", h.GetOwn)
	authed.PUT("/profile", h.Upsert)
	authed.PUT("/preferences", h.UpdatePreferences)
	authed.GET("/profiles/:id", h.GetPublic)

	authed.POST("/profile/photos", middleware.RateLimitByUser(60, 30), h.AddPhoto)
	authed.PUT("/profile/photos/order", h.ReorderPhotos)
	authed.DELETE("/profile/photos/:id", h.DeletePhoto)
	authed.GET("/photos/:id", h.ServePhoto)
}
