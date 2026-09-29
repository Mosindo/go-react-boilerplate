package matching

import (
	"example.com/api/internal/platform/deps"
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r gin.IRouter, pool *pgxpool.Pool, d deps.Common) {
	h := NewHandler(NewService(NewRepository(pool), d.Signer, d.Notifier, d.Publisher))
	limit := ratelimit.ByUser(d.ActionLimiter)
	r.POST("/swipes", d.RequireUser, limit, h.Swipe)
	r.DELETE("/matches/:matchId", d.RequireUser, limit, h.Unmatch)
}
