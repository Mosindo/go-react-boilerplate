package discover

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.GET("/discover", requireUser, middleware.NewLimiter(120, time.Minute).ByUser(), h.List)
}
