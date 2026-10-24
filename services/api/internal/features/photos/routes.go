package photos

import (
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, uploads *ratelimit.Limiter) {
	r.POST("/me/photos", requireUser, uploads.ByUser(), h.Upload)
	r.PUT("/me/photos/order", requireUser, h.Reorder)
	r.PUT("/me/photos/:photoId", requireUser, uploads.ByUser(), h.Replace)
	r.DELETE("/me/photos/:photoId", requireUser, h.Delete)
	r.GET("/photos/:photoId", requireUser, h.Serve)
}
