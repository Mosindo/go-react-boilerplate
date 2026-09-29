package auth

import (
	"time"

	"example.com/api/internal/platform/deps"
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RegisterRoutes wires /auth/*. GET /me lives in the account feature.
func RegisterRoutes(r gin.IRouter, pool *pgxpool.Pool, d deps.Common) {
	registerRoutes(r, pool, d, nil)
}

func registerRoutes(r gin.IRouter, pool *pgxpool.Pool, d deps.Common, spawn func(func())) {
	svc := NewService(NewPGRepository(pool), d.JWTSecret, d.Mailer, d.Config.AppBaseURL)
	h := NewHandler(svc, spawn)

	g := r.Group("/auth", ratelimit.ByIP(d.AuthLimiter))
	// Password reset requests trigger email and are abuse-prone: stricter dedicated budget.
	forgotLimit := ratelimit.ByIP(ratelimit.New(5, time.Minute))

	g.POST("/register", h.Register)
	g.POST("/login", h.Login)
	g.POST("/refresh", h.Refresh)
	g.POST("/logout", h.Logout)
	g.POST("/forgot", forgotLimit, h.Forgot)
	g.POST("/reset", h.Reset)
}
