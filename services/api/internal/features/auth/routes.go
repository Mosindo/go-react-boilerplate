package auth

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, authLimiter *middleware.RateLimiter) {
	limited := authLimiter.ByIP()
	r.POST("/auth/register", limited, handler.Register)
	r.POST("/auth/login", limited, handler.Login)
	r.POST("/auth/refresh", handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/forgot-password", limited, handler.ForgotPassword)
	r.POST("/auth/reset-password", limited, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, handler.DeleteAccount)
}
