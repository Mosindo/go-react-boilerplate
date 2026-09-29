package discovery

import (
	"example.com/api/internal/platform/deps"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r gin.IRouter, pool *pgxpool.Pool, d deps.Common) {
	h := NewHandler(NewService(NewRepository(pool), d.Signer, DefaultRanker{}))
	r.GET("/discover", d.RequireUser, h.Discover)
}
