package auth

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc, authLimit *middleware.RateLimiter) {
	limited := authLimit.ByIP()
	r.POST("/auth/register", limited, h.Register)
	r.POST("/auth/login", limited, h.Login)
	r.POST("/auth/refresh", limited, h.Refresh)
	r.POST("/auth/logout", limited, h.Logout)
	r.POST("/auth/forgot-password", limited, h.ForgotPassword)
	r.POST("/auth/reset-password", limited, h.ResetPassword)

	r.GET("/me", requireUser, h.Me)
	r.POST("/me/password", requireUser, limited, h.ChangePassword)
	r.DELETE("/me", requireUser, limited, h.DeleteAccount)
}
