package auth

import (
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc) {
	// Per-IP limits slow credential stuffing and recovery-code guessing.
	loginLimit := middleware.NewLimiter(10, time.Minute).ByIP()
	registerLimit := middleware.NewLimiter(10, time.Hour).ByIP()
	recoveryLimit := middleware.NewLimiter(10, 15*time.Minute).ByIP()
	refreshLimit := middleware.NewLimiter(60, time.Minute).ByIP()

	r.POST("/auth/register", registerLimit, handler.Register)
	r.POST("/auth/login", loginLimit, handler.Login)
	r.POST("/auth/refresh", refreshLimit, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/forgot", recoveryLimit, handler.Forgot)
	r.POST("/auth/reset", recoveryLimit, handler.Reset)
	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, middleware.NewLimiter(5, time.Hour).ByUser(), handler.DeleteAccount)
}
