// Package app wires platform services and features into the single Gin engine.
package app

import (
	"time"

	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	discoveryfeature "example.com/api/internal/features/discovery"
	matchesfeature "example.com/api/internal/features/matches"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	realtimefeature "example.com/api/internal/features/realtime"
	safetyfeature "example.com/api/internal/features/safety"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Deps struct {
	DB             *pgxpool.Pool
	JWTSecret      []byte
	Store          storage.Store
	Mailer         mailer.Mailer
	AllowedOrigins []string
	// BcryptCost lowers the password hashing cost; tests only (0 keeps the default).
	BcryptCost int
}

// Services exposes the few services other binaries (seed, tests) need.
type Services struct {
	Auth     *authfeature.Service
	Photos   *photosfeature.Service
	Profiles *profilesfeature.Service
}

func New(d Deps) (*gin.Engine, Services) {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(d.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	hub := realtime.NewHub()

	authRepo := authfeature.NewPGRepository(d.DB)
	sessions := middleware.NewCachedSessions(authRepo, 15*time.Second)
	requireUser := middleware.RequireUser(d.JWTSecret, sessions)

	var authOpts []authfeature.Option
	authOpts = append(authOpts, authfeature.WithInvalidator(sessions))
	if d.BcryptCost > 0 {
		authOpts = append(authOpts, authfeature.WithBcryptCost(d.BcryptCost))
	}
	authService := authfeature.NewService(authRepo, d.JWTSecret, d.Mailer, d.Store, authOpts...)

	photosService := photosfeature.NewService(photosfeature.NewPGRepository(d.DB), d.Store, d.JWTSecret)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(d.DB), photosService)
	matchesService := matchesfeature.NewService(matchesfeature.NewPGRepository(d.DB), profilesService, hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(d.DB), hub)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(d.DB), profilesService)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(d.DB), hub)
	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(d.DB), profilesService, profilesService)

	r.GET("/health", platformhandlers.NewHealthHandler(d.DB))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser)
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser)
	matchesfeature.RegisterRoutes(r, matchesfeature.NewHandler(matchesService), requireUser)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser)
	realtimefeature.RegisterRoutes(r, realtimefeature.NewHandler(hub, d.JWTSecret, sessions, d.AllowedOrigins), requireUser)

	return r, Services{Auth: authService, Photos: photosService, Profiles: profilesService}
}
