// Package server wires configuration, platform services and feature modules into one Gin engine.
package server

import (
	"fmt"

	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	discoveryfeature "example.com/api/internal/features/discovery"
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
	"time"
)

type Options struct {
	DB             *pgxpool.Pool
	JWTSecret      []byte
	AllowedOrigins []string
	TrustedProxies []string
	UploadDir      string
	Mailer         mailer.Mailer
}

// App is the assembled HTTP application.
type App struct {
	Engine *gin.Engine
	Hub    *realtime.Hub
}

func New(o Options) (*App, error) {
	store, err := storage.NewLocal(o.UploadDir)
	if err != nil {
		return nil, err
	}
	if o.Mailer == nil {
		o.Mailer = mailer.LogMailer{}
	}

	r := gin.New()
	if err := r.SetTrustedProxies(o.TrustedProxies); err != nil {
		return nil, fmt.Errorf("trusted proxies: %w", err)
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(o.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.BodyLimit(1 << 20))

	hub := realtime.NewHub(o.AllowedOrigins)
	requireUser := middleware.RequireUser(o.JWTSecret)

	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(o.DB))
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(o.DB), store, profilesService)
	authService := authfeature.NewService(authfeature.NewPGRepository(o.DB), o.JWTSecret, o.Mailer, photosService)
	authHandler := authfeature.NewHandler(authService, middleware.NewLimiter(10, 15*time.Minute), middleware.NewLimiter(3, 15*time.Minute))
	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(o.DB), profilesService, hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(o.DB), profilesService, hub)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(o.DB))
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(o.DB), hub)

	r.GET("/health", platformhandlers.NewHealthHandler(o.DB))
	authfeature.RegisterRoutes(r, authHandler, requireUser)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser)
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser)
	r.POST("/ws/ticket", requireUser, hub.TicketHandler)
	r.GET("/ws", hub.ServeWS)

	return &App{Engine: r, Hub: hub}, nil
}
