package auth

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

// Limiters groups the abuse protections applied to authentication routes.
type Limiters struct {
	Credentials *middleware.RateLimiter
	Recovery    *middleware.RateLimiter
}

func DefaultLimiters() Limiters {
	return Limiters{
		Credentials: middleware.NewRateLimiter(20, time.Minute, 10),
		Recovery:    middleware.NewRateLimiter(5, 15*time.Minute, 5),
	}
}

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, limiters Limiters) {
	credentials := middleware.RateLimitByIP(limiters.Credentials, "auth.credentials")
	recovery := middleware.RateLimitByIP(limiters.Recovery, "auth.recovery")

	r.POST("/auth/register", credentials, handler.Register)
	r.POST("/auth/login", credentials, handler.Login)
	r.POST("/auth/refresh", handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", recovery, handler.ForgotPassword)
	r.POST("/auth/password/reset", recovery, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.POST("/me/password", requireUser, credentials, handler.ChangePassword)
	r.DELETE("/me", requireUser, credentials, handler.DeleteAccount)
}
