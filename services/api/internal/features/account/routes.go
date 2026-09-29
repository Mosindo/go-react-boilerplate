package account

import (
	"example.com/api/internal/platform/deps"
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r gin.IRouter, pool *pgxpool.Pool, d deps.Common) {
	h := NewHandler(NewService(NewPGRepository(pool), d.Storage))
	// Password-verifying endpoints are throttled per user.
	sensitive := ratelimit.ByUser(d.AuthLimiter)

	r.GET("/me", d.RequireUser, h.Me)
	r.POST("/me/password", d.RequireUser, sensitive, h.ChangePassword)
	r.DELETE("/me", d.RequireUser, sensitive, h.Delete)
}
