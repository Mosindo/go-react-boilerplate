package auth

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	// Credential endpoints are the favourite target of brute force: keep them tight.
	credentialLimit := middleware.NewLimiter(20, 10*time.Minute).ByIP()
	recoveryLimit := middleware.NewLimiter(8, 15*time.Minute).ByIP()
	refreshLimit := middleware.NewLimiter(60, 10*time.Minute).ByIP()

	r.POST("/auth/register", credentialLimit, handler.Register)
	r.POST("/auth/login", credentialLimit, handler.Login)
	r.POST("/auth/refresh", refreshLimit, handler.Refresh)
	r.POST("/auth/logout", refreshLimit, handler.Logout)
	r.POST("/auth/forgot-password", recoveryLimit, handler.ForgotPassword)
	r.POST("/auth/reset-password", recoveryLimit, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
}
