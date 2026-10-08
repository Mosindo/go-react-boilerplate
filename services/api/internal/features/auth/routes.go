package auth

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	ipLimit := middleware.RateLimitByIP(middleware.NewLimiter(20, time.Minute))
	r.POST("/auth/register", ipLimit, handler.Register)
	r.POST("/auth/login", ipLimit, handler.Login)
	r.POST("/auth/refresh", ipLimit, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", ipLimit, handler.ForgotPassword)
	r.POST("/auth/password/reset", ipLimit, handler.ResetPassword)
	r.POST("/me/password", requireUser, handler.ChangePassword)
	r.DELETE("/me", requireUser, handler.DeleteAccount)
}
