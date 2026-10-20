// Package app wires platform services and feature modules into one Gin router.
package app

import (
	"context"
	"errors"
	"time"

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
)

// Limits are requests per minute (and burst) for each protected area.
type Limits struct {
	AuthPerMinute, AuthBurst     int
	SwipePerMinute, SwipeBurst   int
	SendPerMinute, SendBurst     int
	UploadPerMinute, UploadBurst int
	ReportPerMinute, ReportBurst int
}

func DefaultLimits() Limits {
	return Limits{
		AuthPerMinute: 30, AuthBurst: 30,
		SwipePerMinute: 240, SwipeBurst: 60,
		SendPerMinute: 60, SendBurst: 30,
		UploadPerMinute: 20, UploadBurst: 10,
		ReportPerMinute: 10, ReportBurst: 5,
	}
}

type Deps struct {
	DB             *pgxpool.Pool
	JWTSecret      []byte
	AllowedOrigins []string
	Files          storage.Store
	Mailer         mailer.Mailer
	AppName        string
	Limits         Limits
	// SessionCacheTTL bounds how long a revoked session may keep working (0 = check every request).
	SessionCacheTTL time.Duration
	// TrustedProxies are the proxies whose X-Forwarded-For is honoured for rate limiting; none by default.
	TrustedProxies []string
}

func NewRouter(d Deps) (*gin.Engine, error) {
	if d.DB == nil || len(d.JWTSecret) == 0 || d.Files == nil || d.Mailer == nil {
		return nil, errors.New("app: DB, JWTSecret, Files and Mailer are required")
	}

	r := gin.New()
	if err := r.SetTrustedProxies(d.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(d.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	hub := realtime.NewHub()
	authRepo := authfeature.NewPGRepository(d.DB)
	sessions := middleware.NewCachedSessionChecker(authRepo, d.SessionCacheTTL)
	requireUser := middleware.RequireUser(d.JWTSecret, sessions)

	authLimiter := middleware.NewLimiter(d.Limits.AuthPerMinute, d.Limits.AuthBurst)
	swipeLimiter := middleware.NewLimiter(d.Limits.SwipePerMinute, d.Limits.SwipeBurst)
	sendLimiter := middleware.NewLimiter(d.Limits.SendPerMinute, d.Limits.SendBurst)
	uploadLimiter := middleware.NewLimiter(d.Limits.UploadPerMinute, d.Limits.UploadBurst)
	reportLimiter := middleware.NewLimiter(d.Limits.ReportPerMinute, d.Limits.ReportBurst)

	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(d.DB), hub)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(d.DB))
	authService := authfeature.NewService(authRepo, d.JWTSecret, authfeature.Options{
		Mailer: d.Mailer, Files: d.Files, Conns: hub, AppName: d.AppName,
	})
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(d.DB), d.Files, profilesService)
	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(d.DB), profilesService, notificationsService, hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(d.DB), notificationsService, hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(d.DB), hub)

	r.GET("/health", platformhandlers.NewHealthHandler(d.DB))
	r.GET("/ws", hub.Handler(func(ctx context.Context, token string) (string, error) {
		identity, err := middleware.Authenticate(ctx, d.JWTSecret, sessions, token)
		return identity.UserID, err
	}, d.AllowedOrigins))

	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authLimiter)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, uploadLimiter)
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser, swipeLimiter)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, sendLimiter)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, reportLimiter)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	return r, nil
}
