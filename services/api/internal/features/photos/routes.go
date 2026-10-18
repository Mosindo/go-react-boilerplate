package photos

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, uploads *middleware.RateLimiter) {
	r.GET("/me/photos", requireUser, h.List)
	r.POST("/me/photos", requireUser, uploads.ByUser("photo-upload"), h.Add)
	r.PUT("/me/photos/order", requireUser, h.Reorder)
	r.PUT("/me/photos/:id", requireUser, uploads.ByUser("photo-upload"), h.Replace)
	r.DELETE("/me/photos/:id", requireUser, h.Delete)
	r.GET("/media/photos/:id/:variant", h.Serve)
}
