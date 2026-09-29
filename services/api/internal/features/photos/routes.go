package photos

import (
	"example.com/api/internal/platform/deps"
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r gin.IRouter, pool *pgxpool.Pool, d deps.Common) {
	h := NewHandler(NewService(NewPGRepository(pool), d.Storage, d.Signer))

	r.POST("/me/photos", d.RequireUser, ratelimit.ByUser(d.ActionLimiter), h.Upload)
	r.PUT("/me/photos/order", d.RequireUser, h.Reorder)
	r.DELETE("/me/photos/:id", d.RequireUser, h.Delete)
	// Public: authorised by the HMAC signature in the URL.
	r.GET("/photos/:id/file", h.File)
}
