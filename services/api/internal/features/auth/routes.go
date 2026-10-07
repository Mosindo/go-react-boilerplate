package auth

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	// Brute-force / abuse protection on public credential endpoints.
	credentials := middleware.Limit(20, time.Minute, middleware.ByIP)
	resets := middleware.Limit(5, time.Minute, middleware.ByIP)

	r.POST("/auth/register", credentials, handler.Register)
	r.POST("/auth/login", credentials, handler.Login)
	r.POST("/auth/refresh", credentials, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password-reset/request", resets, handler.RequestPasswordReset)
	r.POST("/auth/password-reset/confirm", resets, handler.ConfirmPasswordReset)
	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, middleware.Limit(5, time.Minute, middleware.ByUser), handler.DeleteAccount)
}
