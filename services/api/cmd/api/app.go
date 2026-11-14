package main

import (
	"context"
	"time"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/features/chat"
	"example.com/api/internal/features/matching"
	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/photos"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/features/safety"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type deps struct {
	pool           *pgxpool.Pool
	jwtSecret      []byte
	store          storage.Storage
	mailer         mailer.Mailer
	hub            *realtime.Hub
	allowedOrigins []string
	trustedProxies []string
}

// limits groups abuse-protection rate limits. They guard against brute
// force and spam; they are generous enough to never act as a product quota.
type limits struct {
	credentials *middleware.RateLimiter
	refresh     *middleware.RateLimiter
	uploads     *middleware.RateLimiter
	swipes      *middleware.RateLimiter
	messages    *middleware.RateLimiter
	reports     *middleware.RateLimiter
}

func defaultLimits() limits {
	return limits{
		credentials: middleware.NewRateLimiter(10, time.Minute),
		refresh:     middleware.NewRateLimiter(30, time.Minute),
		uploads:     middleware.NewRateLimiter(20, 10*time.Minute),
		swipes:      middleware.NewRateLimiter(120, time.Minute),
		messages:    middleware.NewRateLimiter(40, time.Minute),
		reports:     middleware.NewRateLimiter(10, time.Hour),
	}
}

func newRouter(d deps, l limits) *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies(d.trustedProxies)
	r.MaxMultipartMemory = 12 << 20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(d.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.BodyLimit(1 << 20))

	signer := media.NewSigner(d.jwtSecret)

	photosService := photos.NewService(photos.NewPGRepository(d.pool), d.store, signer)
	authService := auth.NewService(auth.NewPGRepository(d.pool), d.jwtSecret, d.mailer, auth.AccountHooks{
		BeforeDelete: func(ctx context.Context, userID string) (func(), error) {
			cleanup, err := photosService.PrepareAccountDeletion(ctx, userID)
			if err != nil {
				return nil, err
			}
			return func() {
				cleanup()
				d.hub.Disconnect(userID)
			}, nil
		},
	})
	profilesService := profiles.NewService(profiles.NewPGRepository(d.pool), signer)
	notificationsService := notifications.NewService(notifications.NewPGRepository(d.pool), d.hub)
	matchingService := matching.NewService(matching.NewPGRepository(d.pool), profilesService, notificationsService, d.hub)
	chatService := chat.NewService(chat.NewPGRepository(d.pool), profilesService, notificationsService, d.hub)
	safetyService := safety.NewService(safety.NewPGRepository(d.pool), d.hub)

	requireUser := middleware.RequireUser(d.jwtSecret, authService.SessionActive)

	r.GET("/health", platformhandlers.NewHealthHandler(d.pool))
	r.GET("/ws", d.hub.Handler(func(ctx context.Context, token string) (string, error) {
		identity, err := middleware.Authenticate(ctx, d.jwtSecret, authService.SessionActive, token)
		return identity.UserID, err
	}, originPatterns(d.allowedOrigins)))

	auth.RegisterRoutes(r, auth.NewHandler(authService), auth.RouteGuards{
		RequireUser: requireUser,
		Credentials: l.credentials.ByIP("credentials"),
		Refresh:     l.refresh.ByIP("refresh"),
	})
	profiles.RegisterRoutes(r, profiles.NewHandler(profilesService), requireUser)
	photos.RegisterRoutes(r, photos.NewHandler(photosService), requireUser, l.uploads.ByUser("uploads"))
	matching.RegisterRoutes(r, matching.NewHandler(matchingService), requireUser, l.swipes.ByUser("swipes"))
	chat.RegisterRoutes(r, chat.NewHandler(chatService), requireUser, l.messages.ByUser("messages"))
	notifications.RegisterRoutes(r, notifications.NewHandler(notificationsService), requireUser)
	safety.RegisterRoutes(r, safety.NewHandler(safetyService), requireUser, l.reports.ByUser("reports"))
	return r
}

// originPatterns converts ALLOWED_ORIGINS (full origins) into host
// patterns understood by the WebSocket origin check.
func originPatterns(origins []string) []string {
	out := make([]string, 0, len(origins))
	for _, o := range origins {
		for _, prefix := range []string{"https://", "http://"} {
			if len(o) > len(prefix) && o[:len(prefix)] == prefix {
				o = o[len(prefix):]
			}
		}
		out = append(out, o)
	}
	return out
}
