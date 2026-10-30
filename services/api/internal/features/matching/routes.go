package matching

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/discover", requireUser, middleware.RateLimit(120, time.Minute), h.Discover)
	r.POST("/swipes", requireUser, middleware.RateLimit(240, time.Minute), h.Swipe)
	r.DELETE("/matches/:id", requireUser, h.Unmatch)
}
