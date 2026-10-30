package auth

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	// Credential endpoints are the main brute-force target: 10 attempts/minute/IP.
	limited := r.Group("/auth", middleware.RateLimit(10, time.Minute))
	limited.POST("/register", handler.Register)
	limited.POST("/login", handler.Login)
	limited.POST("/forgot", handler.ForgotPassword)
	limited.POST("/reset", handler.ResetPassword)

	r.POST("/auth/refresh", middleware.RateLimit(60, time.Minute), handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.GET("/me", requireUser, handler.Me)
	r.PUT("/me/password", requireUser, middleware.RateLimit(5, time.Minute), handler.ChangePassword)
}
