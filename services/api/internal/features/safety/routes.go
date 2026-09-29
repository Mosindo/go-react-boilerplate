package safety

import (
	"example.com/api/internal/platform/deps"
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r gin.IRouter, pool *pgxpool.Pool, d deps.Common) {
	h := NewHandler(NewService(NewRepository(pool), d.Publisher))
	limit := ratelimit.ByUser(d.ActionLimiter)
	r.POST("/blocks", d.RequireUser, limit, h.Block)
	r.GET("/blocks", d.RequireUser, h.List)
	r.DELETE("/blocks/:userId", d.RequireUser, limit, h.Unblock)
	r.POST("/reports", d.RequireUser, limit, h.Report)
}
