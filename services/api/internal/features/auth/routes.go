package auth

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts the auth routes. strict is a tight rate limiter applied
// to credential-guessing surfaces (login, register, password recovery).
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, strict *middleware.RateLimiter) {
	limit := middleware.RateLimit(strict)
	r.POST("/auth/register", limit, handler.Register)
	r.POST("/auth/login", limit, handler.Login)
	r.POST("/auth/refresh", limit, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/forgot-password", limit, handler.ForgotPassword)
	r.POST("/auth/reset-password", limit, handler.ResetPassword)
	r.POST("/auth/change-password", requireUser, limit, handler.ChangePassword)
	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, limit, handler.DeleteAccount)
}
