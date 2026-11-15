// Package app is the composition root: it wires platform services and
// feature modules into a single Gin engine.
package app

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
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Options struct {
	DB             *pgxpool.Pool
	JWTSecret      []byte
	Store          storage.Store
	Mailer         mailer.Mailer
	AllowedOrigins []string
	TrustedProxies []string
	// BcryptCost lets tests use a cheaper cost; 0 means default.
	BcryptCost int
	// DisableRateLimit is only meant for tests that issue many requests.
	DisableRateLimit bool
}

type App struct {
	Router *gin.Engine
	Hub    *realtime.Hub
}

func New(opts Options) (*App, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(opts.TrustedProxies); err != nil {
		return nil, err
	}
	r.MaxMultipartMemory = 12 << 20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(opts.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	hub := realtime.NewHub()
	signer := storage.NewURLSigner(opts.JWTSecret, 6*time.Hour)

	authRepo := auth.NewPGRepository(opts.DB)
	authService := auth.NewService(authRepo, opts.JWTSecret, auth.Deps{
		Mailer: opts.Mailer, Store: opts.Store, Realtime: hub, BcryptCost: opts.BcryptCost,
	})
	requireUser := middleware.RequireUser(opts.JWTSecret, authService)

	notificationService := notifications.NewService(notifications.NewPGRepository(opts.DB), hub)
	profileService := profiles.NewService(profiles.NewPGRepository(opts.DB), signer)
	photoService := photos.NewService(photos.NewPGRepository(opts.DB), opts.Store, signer)
	matchingService := matching.NewService(matching.NewPGRepository(opts.DB), profileService, matching.ScoreRanker{}, notificationService, hub)
	chatService := chat.NewService(chat.NewPGRepository(opts.DB), profileService, notificationService, hub)
	safetyService := safety.NewService(safety.NewPGRepository(opts.DB), hub)

	limit := func(burst int, per time.Duration, scope string, key ratelimit.KeyFunc) gin.HandlerFunc {
		if opts.DisableRateLimit {
			return func(c *gin.Context) { c.Next() }
		}
		return ratelimit.Middleware(ratelimit.New(burst, per), scope, key)
	}

	r.GET("/health", platformhandlers.NewHealthHandler(opts.DB))
	r.GET("/ws", hub.Handler(func(ctx context.Context, token string) (string, error) {
		claims, err := middleware.Authenticate(ctx, opts.JWTSecret, authService, token)
		return claims.UserID, err
	}))

	auth.RegisterRoutes(r, auth.NewHandler(authService), requireUser, auth.Limits{
		Credentials: limit(10, time.Minute, "credentials", ratelimit.ByIP),
		Recovery:    limit(5, time.Hour, "recovery", ratelimit.ByIP),
	})
	profiles.RegisterRoutes(r, profiles.NewHandler(profileService), requireUser)
	photos.RegisterRoutes(r, photos.NewHandler(photoService, opts.Store, signer), requireUser,
		limit(30, time.Hour, "uploads", ratelimit.ByUser))
	// Likes are unlimited for users; the bucket only stops scripted abuse.
	matching.RegisterRoutes(r, matching.NewHandler(matchingService), requireUser,
		limit(120, time.Minute, "swipes", ratelimit.ByUser))
	chat.RegisterRoutes(r, chat.NewHandler(chatService), requireUser,
		limit(40, time.Minute, "messages", ratelimit.ByUser))
	safety.RegisterRoutes(r, safety.NewHandler(safetyService), requireUser,
		limit(10, time.Hour, "reports", ratelimit.ByUser))
	notifications.RegisterRoutes(r, notifications.NewHandler(notificationService), requireUser)

	return &App{Router: r, Hub: hub}, nil
}
