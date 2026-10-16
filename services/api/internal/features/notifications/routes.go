package notifications

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	limit := middleware.NewLimiter(240, time.Minute).ByUser()

	r.GET("/notifications", requireUser, limit, handler.List)
	r.POST("/notifications/read-all", requireUser, limit, handler.MarkAllRead)
	r.POST("/notifications/:notificationId/read", requireUser, limit, handler.MarkRead)
}
