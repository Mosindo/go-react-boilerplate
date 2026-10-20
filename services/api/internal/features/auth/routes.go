package auth

import (
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts the public credential endpoints behind authLimiter
// (per IP) and the account endpoints behind requireUser.
func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, authLimiter *middleware.Limiter) {
	public := r.Group("/auth", middleware.RateLimitByIP(authLimiter))
	public.POST("/register", handler.Register)
	public.POST("/login", handler.Login)
	public.POST("/refresh", handler.Refresh)
	public.POST("/logout", handler.Logout)
	public.POST("/password-reset/request", handler.RequestPasswordReset)
	public.POST("/password-reset/confirm", handler.ConfirmPasswordReset)

	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, middleware.RateLimitByIP(authLimiter), handler.DeleteAccount)
}
