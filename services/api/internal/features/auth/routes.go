package auth

import "github.com/gin-gonic/gin"

// RouteLimits are rate limiting middlewares for abuse-sensitive endpoints.
type RouteLimits struct {
	Register gin.HandlerFunc
	Login    gin.HandlerFunc
	Recovery gin.HandlerFunc
}

func passthrough(c *gin.Context) { c.Next() }

func RegisterRoutes(r gin.IRouter, handler *Handler, requireUser gin.HandlerFunc, limits ...RouteLimits) {
	l := RouteLimits{Register: passthrough, Login: passthrough, Recovery: passthrough}
	if len(limits) > 0 {
		l = limits[0]
	}
	r.POST("/auth/register", l.Register, handler.Register)
	r.POST("/auth/login", l.Login, handler.Login)
	r.POST("/auth/refresh", l.Login, handler.Refresh)
	r.POST("/auth/logout", handler.Logout)
	r.POST("/auth/password/forgot", l.Recovery, handler.ForgotPassword)
	r.POST("/auth/password/reset", l.Recovery, handler.ResetPassword)
	r.GET("/me", requireUser, handler.Me)
	r.DELETE("/me", requireUser, l.Recovery, handler.DeleteAccount)
}
