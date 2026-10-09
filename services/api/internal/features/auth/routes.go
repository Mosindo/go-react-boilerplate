package auth

import (
	"time"

	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes mounts the auth endpoints. Credential endpoints are rate limited
// per client IP to slow down credential stuffing and code guessing.
func RegisterRoutes(r gin.IRouter, h *Handler, requireUser gin.HandlerFunc) {
	credentials := ratelimit.ByIP("auth-credentials", ratelimit.New(20, time.Minute))
	recovery := ratelimit.ByIP("auth-recovery", ratelimit.New(10, time.Hour))
	register := ratelimit.ByIP("auth-register", ratelimit.New(10, time.Hour))

	r.POST("/auth/register", register, h.Register)
	r.POST("/auth/login", credentials, h.Login)
	r.POST("/auth/refresh", credentials, h.Refresh)
	r.POST("/auth/logout", h.Logout)
	r.POST("/auth/password/forgot", recovery, h.ForgotPassword)
	r.POST("/auth/password/reset", recovery, h.ResetPassword)
	r.POST("/me/password", requireUser, credentials, h.ChangePassword)
	r.DELETE("/me", requireUser, credentials, h.DeleteAccount)
}
