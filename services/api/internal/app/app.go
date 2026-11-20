// Package app is the composition root: it wires platform services and
// features into a single Gin engine. cmd/api and the integration tests both
// build the API through New, so tests exercise the real routing.
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
	"example.com/api/internal/platform/mediaurl"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Limits configures abuse protection. Values are per API instance.
type Limits struct {
	GlobalPerIP rate
	Auth        rate
	Swipes      rate
	Messages    rate
	Uploads     rate
	Reports     rate
}

type rate struct {
	Count  int
	Window time.Duration
	Burst  int
}

// DefaultLimits are generous for humans and tight for scripts. They never
// cap likes for product reasons: the swipe limit only stops automation.
func DefaultLimits() Limits {
	return Limits{
		GlobalPerIP: rate{Count: 600, Window: time.Minute, Burst: 120},
		Auth:        rate{Count: 20, Window: time.Minute, Burst: 10},
		Swipes:      rate{Count: 120, Window: time.Minute, Burst: 60},
		Messages:    rate{Count: 60, Window: time.Minute, Burst: 30},
		Uploads:     rate{Count: 30, Window: 10 * time.Minute, Burst: 10},
		Reports:     rate{Count: 10, Window: time.Hour, Burst: 5},
	}
}

func (r rate) limiter() *middleware.RateLimiter {
	return middleware.NewRateLimiter(r.Count, r.Window, r.Burst)
}

type Options struct {
	Mailer mailer.Mailer
	Store  storage.Store
	Limits Limits
}

type App struct {
	Engine *gin.Engine
	Hub    *realtime.Hub
}

func New(cfg config.Config, pool *pgxpool.Pool, opts Options) (*App, error) {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	secret := []byte(cfg.JWTSecret)

	store := opts.Store
	if store == nil {
		local, err := storage.NewLocalStore(cfg.UploadDir)
		if err != nil {
			return nil, err
		}
		store = local
	}
	mail := opts.Mailer
	if mail == nil {
		if cfg.SMTP.Enabled() {
			mail = &mailer.SMTPMailer{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
		} else {
			mail = &mailer.DevMailer{Production: cfg.IsProduction()}
		}
	}
	limits := opts.Limits
	if limits.Auth.Count == 0 {
		limits = DefaultLimits()
	}

	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.MaxMultipartMemory = 12 << 20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.RateLimit(limits.GlobalPerIP.limiter(), middleware.KeyByIP))

	hub := realtime.NewHub()
	signer := mediaurl.NewSigner(secret)

	authRepo := auth.NewPGRepository(pool)
	requireUser := middleware.RequireUser(secret, authRepo)
	byUser := func(l rate) gin.HandlerFunc { return middleware.RateLimit(l.limiter(), middleware.KeyByUser) }

	notificationsService := notifications.NewService(notifications.NewPGRepository(pool), hub)
	profilesService := profiles.NewService(profiles.NewPGRepository(pool), signer)
	photosService := photos.NewService(photos.NewPGRepository(pool), store, signer)
	authService := auth.NewService(authRepo, secret, mail, photosService, hub)
	matchingService := matching.NewService(matching.NewPGRepository(pool), profilesService, notificationsService, hub, matching.HeuristicRanker{})
	chatService := chat.NewService(chat.NewPGRepository(pool), profilesService, notificationsService, hub)
	safetyService := safety.NewService(safety.NewPGRepository(pool), hub)

	r.GET("/health", platformhandlers.NewHealthHandler(pool))
	auth.RegisterRoutes(r, auth.NewHandler(authService), requireUser, middleware.RateLimit(limits.Auth.limiter(), middleware.KeyByIP))
	profiles.RegisterRoutes(r, profiles.NewHandler(profilesService), requireUser)
	photos.RegisterRoutes(r, photos.NewHandler(photosService), requireUser, byUser(limits.Uploads))
	matching.RegisterRoutes(r, matching.NewHandler(matchingService), requireUser, byUser(limits.Swipes))
	chat.RegisterRoutes(r, chat.NewHandler(chatService), requireUser, byUser(limits.Messages))
	notifications.RegisterRoutes(r, notifications.NewHandler(notificationsService), requireUser)
	safety.RegisterRoutes(r, safety.NewHandler(safetyService), requireUser, byUser(limits.Reports))
	r.GET("/realtime", realtime.NewHandler(hub, secret, authRepo, cfg.AllowedOrigins).Serve)

	return &App{Engine: r, Hub: hub}, nil
}
