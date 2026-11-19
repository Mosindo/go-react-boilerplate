// Package app is the composition root: it wires platform services and features
// into a single Gin engine. It is shared by cmd/api and the integration tests.
package app

import (
	"log"
	"net/url"
	"time"

	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	matchingfeature "example.com/api/internal/features/matching"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	realtimefeature "example.com/api/internal/features/realtime"
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Deps struct {
	Config  config.Config
	DB      *pgxpool.Pool
	Mailer  mailer.Mailer
	Storage media.Storage
	// BcryptCost overrides the hashing cost (tests only); 0 keeps the default.
	BcryptCost int
}

type App struct {
	Engine *gin.Engine
	Hub    *realtime.Hub
}

// accountHooks cleans up resources living outside the database after account deletion.
type accountHooks struct {
	photos *photosfeature.Service
	hub    *realtime.Hub
}

func (h accountHooks) AccountDeleted(userID string, photoKeys []string) {
	h.photos.RemoveFiles(photoKeys)
	h.hub.Disconnect(userID)
}

func New(d Deps) (*App, error) {
	cfg := d.Config
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	if err := r.SetTrustedProxies(nilIfEmpty(cfg.TrustedProxies)); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.BodyLimit(64 << 10))

	secret := []byte(cfg.JWTSecret)
	hub := realtime.NewHub()
	signer := media.NewSigner(secret)

	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(d.DB), signer)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(d.DB), d.Storage, signer)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(d.DB), profilesService, hub)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(d.DB), profilesService, notificationsService, hub,
		matchingfeature.Config{IncludeDemo: !cfg.IsProduction()})
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(d.DB), profilesService, notificationsService, hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(d.DB), hub)

	var authOpts []authfeature.Option
	if d.BcryptCost > 0 {
		authOpts = append(authOpts, authfeature.WithBcryptCost(d.BcryptCost))
	}
	authService := authfeature.NewService(authfeature.NewPGRepository(d.DB), secret, d.Mailer,
		accountHooks{photos: photosService, hub: hub}, authOpts...)

	requireUser := middleware.RequireUser(secret, authService)
	perIP := func(burst int, per time.Duration) gin.HandlerFunc {
		return middleware.RateLimit(middleware.NewRateLimiter(burst, per), middleware.KeyByIP)
	}
	perUser := func(burst int, per time.Duration) gin.HandlerFunc {
		return middleware.RateLimit(middleware.NewRateLimiter(burst, per), middleware.KeyByUser)
	}

	r.GET("/health", platformhandlers.NewHealthHandler(d.DB))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), authfeature.RouteGuards{
		RequireUser:     requireUser,
		CredentialLimit: perIP(20, time.Minute),
		RecoveryLimit:   perIP(5, 15*time.Minute),
	})
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, perUser(30, time.Hour))
	// Anti-abuse only (scripts/bots): far above any human swiping pace. Likes are never capped.
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, perUser(120, time.Minute))
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, perUser(60, time.Minute))
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, perUser(10, time.Hour))
	realtimefeature.RegisterRoutes(r, realtimefeature.NewHandler(hub, originHosts(cfg.AllowedOrigins)), requireUser)

	return &App{Engine: r, Hub: hub}, nil
}

func nilIfEmpty(v []string) []string {
	if len(v) == 0 {
		return nil
	}
	return v
}

func originHosts(origins []string) []string {
	hosts := make([]string, 0, len(origins))
	for _, o := range origins {
		u, err := url.Parse(o)
		if err != nil || u.Host == "" {
			log.Printf(`{"event":"ignored_origin","origin":%q}`, o)
			continue
		}
		hosts = append(hosts, u.Host)
	}
	return hosts
}
