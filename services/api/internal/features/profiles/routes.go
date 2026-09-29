package profiles

import (
	"example.com/api/internal/platform/deps"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r gin.IRouter, pool *pgxpool.Pool, d deps.Common) {
	h := NewHandler(NewService(NewPGRepository(pool), d.Signer))

	r.GET("/interests", d.RequireUser, h.Interests)
	r.GET("/me/profile", d.RequireUser, h.GetMine)
	r.PUT("/me/profile", d.RequireUser, h.PutMine)
	r.PUT("/me/location", d.RequireUser, h.PutLocation)
	r.GET("/me/preferences", d.RequireUser, h.GetPreferences)
	r.PUT("/me/preferences", d.RequireUser, h.PutPreferences)
	r.GET("/profiles/:userId", d.RequireUser, h.GetPublic)
}
