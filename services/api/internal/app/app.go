// Package app wires configuration, repositories, services and routes into a single Gin engine.
package app

import (
	"net/http"
	"strings"
	"time"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/features/chat"
	"example.com/api/internal/features/discovery"
	"example.com/api/internal/features/matches"
	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/photos"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultBodyLimit = 1 << 20

// Limits are per-minute request budgets (per IP before authentication, per user afterwards).
type Limits struct {
	Auth    int
	Swipe   int
	Message int
	Report  int
	Upload  int
	// API is the general per-user budget applied to every authenticated request.
	API int
}

func DefaultLimits() Limits {
	return Limits{Auth: 30, Swipe: 150, Message: 60, Report: 10, Upload: 30, API: 600}
}

type Options struct {
	Config config.Config
	Pool   *pgxpool.Pool
	Mailer mailer.Mailer
	Limits Limits
	// BcryptCost overrides the password hashing cost (0 = bcrypt default). Only tests lower it.
	BcryptCost int
}

func New(opts Options) *gin.Engine {
	cfg := opts.Config
	secret := []byte(cfg.JWTSecret)
	limits := opts.Limits
	if limits == (Limits{}) {
		limits = DefaultLimits()
	}
	mail := opts.Mailer
	if mail == nil {
		mail = mailer.New(mailer.SMTPConfig{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom,
		})
	}

	r := gin.New()
	// Without trusted proxies X-Forwarded-For is ignored, so clients cannot spoof their IP to dodge rate limits.
	_ = r.SetTrustedProxies(cfg.TrustedProxies)
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	bodyLimit := middleware.MaxBody(defaultBodyLimit)
	r.Use(func(c *gin.Context) {
		// Photo uploads carry their own, larger limit.
		if strings.HasPrefix(c.Request.URL.Path, "/photos") && c.Request.Method != http.MethodGet {
			c.Next()
			return
		}
		bodyLimit(c)
	})

	hub := realtime.NewHub()

	profilesRepo := profiles.NewRepository(opts.Pool)
	profilesService := profiles.NewService(profilesRepo)

	notificationsService := notifications.NewService(notifications.NewRepository(opts.Pool), profilesService, hub)
	matchesService := matches.NewService(matches.NewRepository(opts.Pool), profilesService, hub)
	chatService := chat.NewService(chat.NewRepository(opts.Pool), profilesService, hub, notificationsService)
	discoveryService := discovery.NewService(discovery.NewRepository(opts.Pool), profilesService, matchesService, hub, notificationsService)
	photosService := photos.NewService(photos.NewRepository(opts.Pool), profilesService)
	safetyService := safety.NewService(safety.NewRepository(opts.Pool), hub)
	authService := auth.NewService(auth.NewPGRepository(opts.Pool), secret, mail, opts.BcryptCost, chatService.PrepareAccountRemoval)

	requireUser := middleware.RequireUser(secret, authService, middleware.NewRateLimiter(limits.API, time.Minute))
	minute := time.Minute
	authLimit := middleware.NewRateLimiter(limits.Auth, minute).Middleware("auth")
	swipeLimit := middleware.NewRateLimiter(limits.Swipe, minute).Middleware("swipe")
	messageLimit := middleware.NewRateLimiter(limits.Message, minute).Middleware("message")
	reportLimit := middleware.NewRateLimiter(limits.Report, minute).Middleware("report")
	uploadLimit := middleware.NewRateLimiter(limits.Upload, minute).Middleware("upload")

	r.GET("/health", platformhandlers.NewHealthHandler(opts.Pool))
	auth.RegisterRoutes(r, auth.NewHandler(authService), requireUser, authLimit)
	profiles.RegisterRoutes(r, profiles.NewHandler(profilesService), requireUser)
	photos.RegisterRoutes(r, photos.NewHandler(photosService), requireUser, uploadLimit)
	discovery.RegisterRoutes(r, discovery.NewHandler(discoveryService), requireUser, swipeLimit)
	matches.RegisterRoutes(r, matches.NewHandler(matchesService), requireUser)
	chat.RegisterRoutes(r, chat.NewHandler(chatService), chat.NewSocketHandler(hub, secret, authService, cfg.AllowedOrigins), requireUser, messageLimit)
	safety.RegisterRoutes(r, safety.NewHandler(safetyService), requireUser, reportLimit)
	notifications.RegisterRoutes(r, notifications.NewHandler(notificationsService), requireUser)
	return r
}
