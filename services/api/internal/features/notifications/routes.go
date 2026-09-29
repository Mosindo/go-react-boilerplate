package notifications

import (
	"example.com/api/internal/platform/deps"
	"example.com/api/internal/platform/notify"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewNotifier builds the notify.Notifier used by other features (match, message).
func NewNotifier(pool *pgxpool.Pool, pub realtime.Publisher) notify.Notifier {
	return NewService(NewRepository(pool), pub)
}

func RegisterRoutes(r gin.IRouter, pool *pgxpool.Pool, d deps.Common) {
	h := NewHandler(NewService(NewRepository(pool), d.Publisher))
	r.GET("/notifications", d.RequireUser, h.List)
	// Registered before the :id route; Gin resolves the static segment first anyway.
	r.POST("/notifications/read-all", d.RequireUser, h.MarkAllRead)
	r.POST("/notifications/:id/read", d.RequireUser, h.MarkRead)
}
