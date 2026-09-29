// Package deps carries the shared, already-constructed platform services that every feature
// receives in its RegisterRoutes. Features never build these themselves.
package deps

import (
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/notify"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/urlsign"
	"github.com/gin-gonic/gin"
)

type Common struct {
	Config      config.Config
	JWTSecret   []byte
	RequireUser gin.HandlerFunc
	Storage     storage.Storage
	Signer      *urlsign.Signer
	Mailer      mailer.Mailer
	Publisher   realtime.Publisher
	Notifier    notify.Notifier
	// AuthLimiter throttles credential endpoints per IP; ActionLimiter throttles
	// high-frequency authenticated actions (swipes, messages) per user.
	AuthLimiter   *ratelimit.Limiter
	ActionLimiter *ratelimit.Limiter
}
