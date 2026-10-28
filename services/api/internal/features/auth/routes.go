package auth

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	// Per-IP throttles against credential stuffing and code guessing (per-process counters).
	register := middleware.RateLimitByIP(middleware.NewLimiter(10, time.Hour))
	login := middleware.RateLimitByIP(middleware.NewLimiter(20, 5*time.Minute))
	refresh := middleware.RateLimitByIP(middleware.NewLimiter(60, 5*time.Minute))
	forgot := middleware.RateLimitByIP(middleware.NewLimiter(5, 15*time.Minute))
	reset := middleware.RateLimitByIP(middleware.NewLimiter(10, 15*time.Minute))
	sensitive := middleware.RateLimitByUser(middleware.NewLimiter(10, 15*time.Minute))

	r.POST("/auth/register", register, handler.Register)
	r.POST("/auth/login", login, handler.Login)
	r.POST("/auth/refresh", refresh, handler.Refresh)
	r.POST("/auth/logout", refresh, handler.Logout)
	r.POST("/auth/forgot-password", forgot, handler.ForgotPassword)
	r.POST("/auth/reset-password", reset, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.POST("/me/password", requireUser, sensitive, handler.ChangePassword)
	r.DELETE("/me", requireUser, sensitive, handler.DeleteAccount)
}
