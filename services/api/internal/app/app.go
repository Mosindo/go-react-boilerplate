// Package app is the composition root: it wires platform services and
// features into a single Gin router. It is shared by the API binary and the
// HTTP integration tests.
package app

import (
	"log"
	"time"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/features/chat"
	"example.com/api/internal/features/discovery"
	"example.com/api/internal/features/matching"
	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/photos"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/features/safety"
	"example.com/api/internal/platform/authtoken"
	"example.com/api/internal/platform/config"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	Router *gin.Engine
	Hub    *realtime.Hub
}

// Limits groups the anti-abuse rate limiters (never product quotas).
type Limits struct {
	Auth    *ratelimit.Limiter
	Swipe   *ratelimit.Limiter
	Message *ratelimit.Limiter
	Upload  *ratelimit.Limiter
	Report  *ratelimit.Limiter
}

func DefaultLimits() Limits {
	return Limits{
		Auth:    ratelimit.New(10, time.Minute, 10),
		Swipe:   ratelimit.New(120, time.Minute, 60),
		Message: ratelimit.New(60, time.Minute, 30),
		Upload:  ratelimit.New(20, time.Minute, 10),
		Report:  ratelimit.New(20, time.Hour, 5),
	}
}

func DisabledLimits() Limits {
	return Limits{
		Auth:    ratelimit.Disabled(),
		Swipe:   ratelimit.Disabled(),
		Message: ratelimit.Disabled(),
		Upload:  ratelimit.Disabled(),
		Report:  ratelimit.Disabled(),
	}
}

type Options struct {
	Limits Limits
	Mailer mailer.Mailer
}

func New(cfg config.Config, pool *pgxpool.Pool) (*App, error) {
	var mail mailer.Mailer
	switch {
	case cfg.SMTP.Enabled():
		mail = mailer.NewSMTPMailer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
	case !cfg.IsProduction():
		mail = mailer.LogMailer{}
	default:
		log.Printf("warning: SMTP is not configured; password reset is disabled")
	}
	limits := DefaultLimits()
	if cfg.AppEnv == config.EnvTest {
		limits = DisabledLimits()
	}
	return NewWithOptions(cfg, pool, Options{Limits: limits, Mailer: mail})
}

func NewWithOptions(cfg config.Config, pool *pgxpool.Pool, opts Options) (*App, error) {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	store, err := storage.NewLocalStore(cfg.UploadDir)
	if err != nil {
		return nil, err
	}

	tokens := authtoken.NewManager([]byte(cfg.JWTSecret))
	signer := media.NewSigner([]byte(cfg.JWTSecret))
	hub := realtime.NewHub(pool, tokens)

	profilesService := profiles.NewService(profiles.NewPGRepository(pool), signer)
	photosService := photos.NewService(photos.NewPGRepository(pool), store, signer)
	notificationsService := notifications.NewService(notifications.NewPGRepository(pool), hub)
	authService := auth.NewService(auth.NewPGRepository(pool), tokens, opts.Mailer, photosService)
	matchingService := matching.NewService(matching.NewPGRepository(pool), profilesService, notificationsService, hub)
	discoveryService := discovery.NewService(discovery.NewPGRepository(pool), profilesService, discovery.DefaultScorer{})
	chatService := chat.NewService(chat.NewPGRepository(pool), profilesService, notificationsService, hub)
	safetyService := safety.NewService(safety.NewPGRepository(pool), profilesService, hub)

	r := gin.New()
	if len(cfg.TrustedProxies) > 0 {
		if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
			return nil, err
		}
	} else {
		// Without explicit proxies, never trust X-Forwarded-For: rate limits
		// must key on the real peer address.
		if err := r.SetTrustedProxies(nil); err != nil {
			return nil, err
		}
	}
	r.MaxMultipartMemory = 12 << 20

	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	requireUser := middleware.RequireUser(tokens, authService)
	authLimit := ratelimit.Middleware(opts.Limits.Auth, "auth", ratelimit.ByIP)
	swipeLimit := ratelimit.Middleware(opts.Limits.Swipe, "swipe", ratelimit.ByUser)
	messageLimit := ratelimit.Middleware(opts.Limits.Message, "message", ratelimit.ByUser)
	uploadLimit := ratelimit.Middleware(opts.Limits.Upload, "upload", ratelimit.ByUser)
	reportLimit := ratelimit.Middleware(opts.Limits.Report, "report", ratelimit.ByUser)

	r.GET("/health", platformhandlers.NewHealthHandler(pool))
	auth.RegisterRoutes(r, auth.NewHandler(authService), requireUser, authLimit)
	profiles.RegisterRoutes(r, profiles.NewHandler(profilesService), requireUser)
	photos.RegisterRoutes(r, photos.NewHandler(photosService), requireUser, uploadLimit)
	discovery.RegisterRoutes(r, discovery.NewHandler(discoveryService), requireUser)
	matching.RegisterRoutes(r, matching.NewHandler(matchingService), requireUser, swipeLimit)
	chat.RegisterRoutes(r, chat.NewHandler(chatService), requireUser, messageLimit)
	notifications.RegisterRoutes(r, notifications.NewHandler(notificationsService), requireUser)
	safety.RegisterRoutes(r, safety.NewHandler(safetyService), requireUser, reportLimit)
	realtime.RegisterRoutes(r, hub, requireUser)

	return &App{Router: r, Hub: hub}, nil
}
