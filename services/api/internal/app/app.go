// Package app is the composition root: it wires platform services and features into one Gin engine.
package app

import (
	"time"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/features/chat"
	"example.com/api/internal/features/matching"
	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/photos"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultBodyLimit = 1 << 20                      // 1 MiB for JSON endpoints
	uploadBodyLimit  = media.MaxUploadBytes + 1<<20 // photo + multipart overhead
)

type Options struct {
	Mailer  mailer.Mailer
	Storage media.Storage
	// DisableRateLimits is only meant for automated tests that create many accounts quickly.
	DisableRateLimits bool
}

type App struct {
	Router *gin.Engine
	Hub    *realtime.Hub
}

func New(cfg config.Config, pool *pgxpool.Pool, opts Options) (*App, error) {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	storage := opts.Storage
	if storage == nil {
		local, err := media.NewLocalStorage(cfg.UploadDir)
		if err != nil {
			return nil, err
		}
		storage = local
	}
	mail := opts.Mailer
	if mail == nil {
		if cfg.SMTP.Enabled() {
			mail = mailer.SMTPMailer{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
		} else {
			mail = mailer.LogMailer{}
		}
	}

	jwtSecret := []byte(cfg.JWTSecret)
	hub := realtime.NewHub()
	signer := media.NewSigner(jwtSecret)

	authService := auth.NewService(auth.NewPGRepository(pool), jwtSecret, mail, storage, hub)
	requireUser := middleware.RequireUser(jwtSecret, authService)

	notificationService := notifications.NewService(notifications.NewPGRepository(pool), hub)
	profileService := profiles.NewService(profiles.NewPGRepository(pool), signer)
	photoService := photos.NewService(photos.NewPGRepository(pool), storage, signer)
	matchingService := matching.NewService(matching.NewPGRepository(pool), profileService, matching.NewScoreRanker(), notificationService, hub)
	chatService := chat.NewService(chat.NewPGRepository(pool), profileService, notificationService, hub)
	safetyService := safety.NewService(safety.NewPGRepository(pool), matchingService)

	limit := func(n int, window time.Duration, key func(*gin.Context) string) gin.HandlerFunc {
		if opts.DisableRateLimits {
			return func(c *gin.Context) { c.Next() }
		}
		return middleware.RateLimit(middleware.NewRateLimiter(n, window), key)
	}

	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.BodyLimit(defaultBodyLimit, map[string]int64{
		"/photos":          uploadBodyLimit,
		"/photos/:photoId": uploadBodyLimit,
	}))

	r.GET("/health", platformhandlers.NewHealthHandler(pool))

	auth.RegisterRoutes(r, auth.NewHandler(authService), requireUser, limit(20, 10*time.Minute, middleware.KeyByIP))
	profiles.RegisterRoutes(r, profiles.NewHandler(profileService), requireUser, limit(20, time.Hour, middleware.KeyByUser))
	photos.RegisterRoutes(r, photos.NewHandler(photoService), requireUser, limit(30, 10*time.Minute, middleware.KeyByUser))
	// Swipes are not capped for product reasons: the limit only stops scripted abuse.
	matching.RegisterRoutes(r, matching.NewHandler(matchingService), requireUser, limit(120, time.Minute, middleware.KeyByUser))
	chat.RegisterRoutes(r, chat.NewHandler(chatService), requireUser, limit(40, time.Minute, middleware.KeyByUser))
	safety.RegisterRoutes(r, safety.NewHandler(safetyService), requireUser, limit(20, time.Hour, middleware.KeyByUser))
	notifications.RegisterRoutes(r, notifications.NewHandler(notificationService), requireUser)

	realtime.NewHandler(hub, realtime.NewTickets(30*time.Second), cfg.AllowedOrigins, nil).
		RegisterRoutes(r, requireUser)

	return &App{Router: r, Hub: hub}, nil
}
