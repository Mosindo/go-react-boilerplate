package auth

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	// Credential endpoints are the brute-force surface: tight per-IP budgets.
	credentials := middleware.RateLimitByIP(20, time.Minute)
	recovery := middleware.RateLimitByIP(5, 15*time.Minute)
	refresh := middleware.RateLimitByIP(60, time.Minute)

	r.POST("/auth/register", credentials, handler.Register)
	r.POST("/auth/login", credentials, handler.Login)
	r.POST("/auth/refresh", refresh, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", recovery, handler.ForgotPassword)
	r.POST("/auth/password/reset", recovery, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
}
