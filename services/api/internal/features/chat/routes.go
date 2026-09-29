package chat

import (
	"example.com/api/internal/platform/deps"
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(r gin.IRouter, pool *pgxpool.Pool, d deps.Common) {
	h := NewHandler(NewService(NewRepository(pool), d.Signer, d.Publisher, d.Notifier))
	r.GET("/conversations", d.RequireUser, h.ListConversations)
	r.GET("/conversations/:id/messages", d.RequireUser, h.ListMessages)
	r.POST("/conversations/:id/messages", d.RequireUser, ratelimit.ByUser(d.ActionLimiter), h.SendMessage)
	r.POST("/conversations/:id/read", d.RequireUser, h.MarkRead)
	r.DELETE("/conversations/:id", d.RequireUser, h.Hide)
}
