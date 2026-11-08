package main

import (
	"example.com/api/internal/features/auth"
	"example.com/api/internal/features/chat"
	"example.com/api/internal/features/discovery"
	"example.com/api/internal/features/matches"
	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/photos"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/features/safety"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Limits configures the in-memory rate limiters (requests per minute and burst size).
type Limits struct {
	Public, Swipes, Messages, Uploads, Reports [2]int
}

func DefaultLimits() Limits {
	return Limits{
		Public:   [2]int{60, 20},
		Swipes:   [2]int{180, 60},
		Messages: [2]int{90, 30},
		Uploads:  [2]int{20, 10},
		Reports:  [2]int{10, 5},
	}
}

type Deps struct {
	DB             *pgxpool.Pool
	JWTSecret      []byte
	Storage        storage.Storage
	Mailer         mailer.Mailer
	AllowedOrigins []string
	TrustedProxies []string
	Limits         Limits
}

// NewRouter wires every feature (handler -> service -> repository) onto a single Gin engine.
func NewRouter(d Deps) (*gin.Engine, error) {
	r := gin.New()
	// Without an explicit proxy list the client IP is the socket peer, so X-Forwarded-For cannot
	// be spoofed to evade per-IP rate limits.
	if err := r.SetTrustedProxies(d.TrustedProxies); err != nil {
		return nil, err
	}
	r.MaxMultipartMemory = 8 << 20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(d.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	hub := realtime.NewHub()

	authRepo := auth.NewPGRepository(d.DB)
	authService := auth.NewService(authRepo, d.JWTSecret, d.Mailer, d.Storage, hub)
	requireUser := middleware.RequireUser(d.JWTSecret, authService.SessionActive)

	profilesService := profiles.NewService(profiles.NewPGRepository(d.DB))
	notificationsService := notifications.NewService(notifications.NewPGRepository(d.DB), profilesService, hub)
	photosService := photos.NewService(photos.NewPGRepository(d.DB), d.Storage, profilesService)
	matchesService := matches.NewService(matches.NewPGRepository(d.DB), profilesService, profilesService, notificationsService, hub)
	chatService := chat.NewService(chat.NewPGRepository(d.DB), profilesService, notificationsService, hub)
	discoveryService := discovery.NewService(d.DB, profilesService, profilesService)

	limiter := func(l [2]int) *middleware.Limiter { return middleware.NewLimiter(l[0], l[1]) }

	r.GET("/health", platformhandlers.NewHealthHandler(d.DB))
	r.GET("/ws", hub.Handler(d.JWTSecret, authService.SessionActive, d.AllowedOrigins))

	auth.RegisterRoutes(r, auth.NewHandler(authService), requireUser, limiter(d.Limits.Public))
	profiles.RegisterRoutes(r, profiles.NewHandler(profilesService), requireUser)
	photos.RegisterRoutes(r, photos.NewHandler(photosService), requireUser, limiter(d.Limits.Uploads))
	discovery.RegisterRoutes(r, discovery.NewHandler(discoveryService), requireUser)
	matches.RegisterRoutes(r, matches.NewHandler(matchesService), requireUser, limiter(d.Limits.Swipes))
	chat.RegisterRoutes(r, chat.NewHandler(chatService), requireUser, limiter(d.Limits.Messages))
	notifications.RegisterRoutes(r, notifications.NewHandler(notificationsService), requireUser)
	safety.RegisterRoutes(r, safety.NewHandler(safety.NewService(d.DB)), requireUser, limiter(d.Limits.Reports))
	return r, nil
}
