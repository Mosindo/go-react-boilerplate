// Package app wires the HTTP server: platform middleware plus every feature module.
package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	discoveryfeature "example.com/api/internal/features/discovery"
	matchesfeature "example.com/api/internal/features/matches"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	safetyfeature "example.com/api/internal/features/safety"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps are the infrastructure pieces the router needs.
type Deps struct {
	DB             *pgxpool.Pool
	JWTSecret      []byte
	Storage        storage.Storage
	Mailer         mail.Mailer
	AllowedOrigins []string
	TrustedProxies []string
	// SessionCacheTTL bounds how long a revoked session may keep working (0 = check every request).
	SessionCacheTTL time.Duration
}

func NewRouter(d Deps) (*gin.Engine, error) {
	r := gin.New()
	// Without an explicit list no proxy is trusted: X-Forwarded-For cannot be spoofed to dodge rate limits.
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

	signer := storage.NewSigner(d.JWTSecret)
	authSvc := authfeature.NewService(authfeature.NewPGRepository(d.DB), d.JWTSecret, d.Mailer, d.Storage)
	validateSession := middleware.CachedValidator(authSvc.SessionActive, d.SessionCacheTTL)
	hub := realtime.NewHub(func(token string) (string, error) {
		userID, sessionID, err := middleware.ParseAccessToken(d.JWTSecret, token)
		if err != nil {
			return "", err
		}
		if ok, err := validateSession(context.Background(), sessionID); err != nil || !ok {
			return "", errors.New("session inactive")
		}
		return userID, nil
	})
	requireUser := middleware.RequireUser(d.JWTSecret, validateSession)

	profilesSvc := profilesfeature.NewService(profilesfeature.NewPGRepository(d.DB), signer)
	photosSvc := photosfeature.NewService(photosfeature.NewPGRepository(d.DB), d.Storage, signer)
	discoverySvc := discoveryfeature.NewService(discoveryfeature.NewPGRepository(d.DB), profilesSvc)
	matchesSvc := matchesfeature.NewService(matchesfeature.NewPGRepository(d.DB), signer, hub)
	chatSvc := chatfeature.NewService(chatfeature.NewPGRepository(d.DB), signer, hub)
	notificationsSvc := notificationsfeature.NewService(notificationsfeature.NewPGRepository(d.DB), signer)
	safetySvc := safetyfeature.NewService(safetyfeature.NewPGRepository(d.DB), signer, hub)

	r.GET("/health", platformhandlers.NewHealthHandler(d.DB))
	r.GET("/ws", hub.Handle)
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authSvc), requireUser)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesSvc), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosSvc), requireUser)
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoverySvc), requireUser)
	matchesfeature.RegisterRoutes(r, matchesfeature.NewHandler(matchesSvc), requireUser)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatSvc), requireUser)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsSvc), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetySvc), requireUser)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found", "code": "not_found"})
	})
	return r, nil
}
