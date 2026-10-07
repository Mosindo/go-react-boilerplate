package photos

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	upload := middleware.Limit(30, time.Hour, middleware.ByUser)

	r.GET("/photos", requireUser, h.List)
	r.POST("/photos", requireUser, upload, h.Upload)
	r.PUT("/photos/order", requireUser, h.Reorder)
	r.PUT("/photos/:id", requireUser, upload, h.Replace)
	r.DELETE("/photos/:id", requireUser, h.Delete)
	// Public by design: authorisation is the HMAC-signed, expiring URL.
	r.GET("/media/:id", middleware.Limit(600, time.Minute, middleware.ByIP), h.Serve)
}
