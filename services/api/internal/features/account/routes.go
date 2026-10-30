package account

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	r.DELETE("/me", requireUser, middleware.RateLimit(5, time.Minute), h.Delete)
}
