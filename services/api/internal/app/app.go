// Package app assembles the HTTP application: it wires every feature
// (handler -> service -> repository) onto a single Gin engine.
package app

import (
	"time"

	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	matchingfeature "example.com/api/internal/features/matching"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	safetyfeature "example.com/api/internal/features/safety"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Options struct {
	Pool           *pgxpool.Pool
	JWTSecret      []byte
	Storage        storage.Storage
	Mailer         mailer.Mailer
	AllowedOrigins []string
	TrustedProxies []string
	RateLimit      bool
}

func NewRouter(o Options) *gin.Engine {
	r := gin.New()
	// Without an explicit list, client IPs come from the TCP peer only: a client
	// cannot spoof X-Forwarded-For to dodge rate limits.
	_ = r.SetTrustedProxies(o.TrustedProxies)
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(o.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.MaxJSONBody(1 << 20))

	limit := func(max int, per time.Duration) *middleware.RateLimiter {
		return middleware.NewRateLimiter(max, per, o.RateLimit)
	}
	requireUser := middleware.RequireUser(o.JWTSecret)

	hub := realtime.NewHub()
	gateway := realtime.NewGateway(hub, o.JWTSecret)

	photosService := photosfeature.NewService(photosfeature.NewPGRepository(o.Pool), o.Storage, photosfeature.NewSigner(o.JWTSecret))
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(o.Pool), photosService)

	authService := authfeature.NewService(authfeature.NewPGRepository(o.Pool), o.JWTSecret, o.Mailer, o.Storage)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(o.Pool), profilesService, matchingfeature.ProximityInterestRanker{}, hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(o.Pool), photosService, hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(o.Pool), hub)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(o.Pool))

	r.GET("/health", platformhandlers.NewHealthHandler(o.Pool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authfeature.Limiters{
		Auth:    limit(20, time.Minute),
		Recover: limit(5, 15*time.Minute),
	})
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, limit(30, time.Hour))
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, limit(120, time.Minute))
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, limit(60, time.Minute))
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, limit(10, time.Hour))
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	gateway.RegisterRoutes(r, requireUser)
	return r
}
