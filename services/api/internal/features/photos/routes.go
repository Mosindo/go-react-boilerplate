package photos

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	uploadLimit := middleware.RateLimitByUser(middleware.NewLimiter(40, time.Hour))
	r.POST("/me/photos", requireUser, uploadLimit, h.Upload)
	r.PUT("/me/photos/order", requireUser, h.Reorder)
	r.PUT("/me/photos/:id", requireUser, uploadLimit, h.Replace)
	r.DELETE("/me/photos/:id", requireUser, h.Delete)
	r.GET("/photos/:id", requireUser, h.Serve)
}
