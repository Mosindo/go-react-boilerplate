package auth

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	// Credential endpoints are the favourite target of brute force: strict per-IP limits.
	authLimit := middleware.NewRateLimiter(20, time.Minute).ByIP()
	recoveryLimit := middleware.NewRateLimiter(5, 15*time.Minute).ByIP()

	r.POST("/auth/register", authLimit, handler.Register)
	r.POST("/auth/login", authLimit, handler.Login)
	r.POST("/auth/refresh", authLimit, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/forgot-password", recoveryLimit, handler.ForgotPassword)
	r.POST("/auth/reset-password", recoveryLimit, handler.ResetPassword)

	r.GET("/me", requireUser, handler.Me)
	r.POST("/me/password", requireUser, authLimit, handler.ChangePassword)
	r.DELETE("/me", requireUser, authLimit, handler.DeleteAccount)
}
