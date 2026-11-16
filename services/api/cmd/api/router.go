package main

import (
	"time"

	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	discoveryfeature "example.com/api/internal/features/discovery"
	matchingfeature "example.com/api/internal/features/matching"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/imaging"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps are the infrastructure dependencies; tests inject their own.
type Deps struct {
	Pool    *pgxpool.Pool
	Storage storage.Storage
	Mailer  mailer.Mailer
	Hub     *realtime.Hub
}

// RateLimits groups the limiter settings so tests can relax them.
type RateLimits struct {
	Auth     *ratelimit.Limiter
	Swipes   *ratelimit.Limiter
	Messages *ratelimit.Limiter
	Uploads  *ratelimit.Limiter
	Reports  *ratelimit.Limiter
	// Location changes are throttled to make distance trilateration impractical.
	Location *ratelimit.Limiter
	Realtime *ratelimit.Limiter
}

func defaultRateLimits() RateLimits {
	return RateLimits{
		Auth:     ratelimit.New(10, time.Minute, 10),
		Swipes:   ratelimit.New(120, time.Minute, 60),
		Messages: ratelimit.New(60, time.Minute, 30),
		Uploads:  ratelimit.New(20, time.Minute, 5),
		Reports:  ratelimit.New(10, time.Hour, 5),
		Location: ratelimit.New(10, time.Hour, 5),
		Realtime: ratelimit.New(30, time.Minute, 10),
	}
}

func buildRouter(cfg config.Config, deps Deps) *gin.Engine {
	return buildRouterWithLimits(cfg, deps, defaultRateLimits())
}

func buildRouterWithLimits(cfg config.Config, deps Deps, limits RateLimits) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	if deps.Hub == nil {
		deps.Hub = realtime.NewHub()
	}
	relaxedDefault := ratelimit.New(10_000, time.Second, 10_000)
	if limits.Location == nil {
		limits.Location = relaxedDefault
	}
	if limits.Realtime == nil {
		limits.Realtime = relaxedDefault
	}

	r := gin.New()
	// Only trust X-Forwarded-For from explicitly configured proxies, otherwise
	// clients could spoof their IP and bypass rate limits.
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		panic(err)
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.BodyLimit(1<<20, imaging.MaxUploadBytes+(1<<20)))

	secret := []byte(cfg.JWTSecret)
	signer := media.NewSigner(secret, time.Hour)

	authService := authfeature.NewService(authfeature.NewPGRepository(deps.Pool), secret, deps.Mailer, deps.Storage, deps.Hub)
	requireUser := middleware.RequireUser(secret, authService.IsSessionActive)

	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(deps.Pool), signer)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(deps.Pool), deps.Storage, signer)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(deps.Pool), deps.Hub)
	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(deps.Pool), profilesService, discoveryfeature.HeuristicRanker{}, !cfg.IsProduction())
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(deps.Pool), profilesService, profilesService, notificationsService, deps.Hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(deps.Pool), notificationsService, deps.Hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(deps.Pool), deps.Hub)
	realtimeHandler := realtime.NewHandler(deps.Hub, cfg.AllowedOrigins, authService.IsSessionActive)

	byIP := func(l *ratelimit.Limiter, scope string) gin.HandlerFunc {
		return ratelimit.Middleware(l, scope, ratelimit.ByIP)
	}
	byUser := func(l *ratelimit.Limiter, scope string) gin.HandlerFunc {
		return ratelimit.Middleware(l, scope, ratelimit.ByUser)
	}

	r.GET("/health", platformhandlers.NewHealthHandler(deps.Pool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, byIP(limits.Auth, "auth"))
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser, byUser(limits.Location, "location"))
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, byUser(limits.Uploads, "uploads"))
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, byUser(limits.Swipes, "swipes"))
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, byUser(limits.Messages, "messages"))
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, byUser(limits.Reports, "reports"))
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	r.POST("/realtime/ticket", requireUser, byUser(limits.Realtime, "realtime"), realtimeHandler.Ticket)
	r.GET("/realtime", realtimeHandler.Connect)
	return r
}
