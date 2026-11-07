package auth

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, authLimiter *middleware.Limiter) {
	limited := middleware.RateLimitByIP(authLimiter)
	r.POST("/auth/register", limited, h.Register)
	r.POST("/auth/login", limited, h.Login)
	r.POST("/auth/refresh", limited, h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/forgot-password", limited, h.ForgotPassword)
	r.POST("/auth/reset-password", limited, h.ResetPassword)
	r.GET("/me", requireUser, h.Me)
	r.POST("/me/password", requireUser, h.ChangePassword)
	r.DELETE("/me", requireUser, h.DeleteAccount)
}
