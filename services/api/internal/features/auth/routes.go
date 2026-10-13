package auth

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, limiter *middleware.RateLimiter) {
	authLimit := limiter.ByIP("auth", 20, time.Minute)
	recoveryLimit := limiter.ByIP("recovery", 10, 10*time.Minute)

	r.POST("/auth/register", authLimit, h.Register)
	r.POST("/auth/login", authLimit, h.Login)
	r.POST("/auth/refresh", authLimit, h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/password/forgot", recoveryLimit, h.ForgotPassword)
	r.POST("/auth/password/reset", recoveryLimit, h.ResetPassword)
	r.GET("/me", requireUser, h.Me)
	r.DELETE("/me", requireUser, limiter.ByUser("delete-account", 5, time.Hour), h.DeleteAccount)
}
