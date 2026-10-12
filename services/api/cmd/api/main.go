package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	alertsfeature "example.com/api/internal/features/alerts"
	authfeature "example.com/api/internal/features/auth"
	discoveryfeature "example.com/api/internal/features/discovery"
	matchingfeature "example.com/api/internal/features/matching"
	messagingfeature "example.com/api/internal/features/messaging"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mediaurl"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	uploadDir      string
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 30*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		uploadDir:      cfg.UploadDir,
	}
	r, err := setupRouter(a)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		log.Printf("api listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func setupRouter(a *app) (*gin.Engine, error) {
	r := gin.New()
	// Only trust X-Forwarded-For from explicitly configured proxies, otherwise
	// clients could spoof their IP and dodge rate limits.
	if err := r.SetTrustedProxies(a.trustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	hub := realtime.NewHub()
	signer := mediaurl.New(a.jwtSecret)
	storage, err := photosfeature.NewLocalStorage(a.uploadDir)
	if err != nil {
		return nil, err
	}

	// Auth
	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret)
	requireUser := middleware.RequireUser(a.jwtSecret, authService.ValidateSession)

	// Profiles and photos
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), signer)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), storage, signer)
	authService.WithAccountDeletion(photosService.PrepareAccountDeletion)

	// Matching, chat, safety, notifications
	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(a.dbPool), profilesService)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), profilesService, hub)
	messagingService := messagingfeature.NewService(messagingfeature.NewPGRepository(a.dbPool), hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), profilesService, hub)
	alertsService := alertsfeature.NewService(alertsfeature.NewPGRepository(a.dbPool), profilesService)

	// Abuse protection. Likes are never capped per day: these limits only stop bots.
	authLimit := middleware.RateLimit(0.2, 10, nil)                  // per IP
	swipeLimit := middleware.RateLimit(5, 40, middleware.ByUser)     // per user
	sendLimit := middleware.RateLimit(2, 20, middleware.ByUser)      // per user
	uploadLimit := middleware.RateLimit(0.5, 12, middleware.ByUser)  // per user
	reportLimit := middleware.RateLimit(1.0/6, 5, middleware.ByUser) // per user

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authLimit)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, uploadLimit)
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, swipeLimit)
	messagingfeature.RegisterRoutes(r, messagingfeature.NewHandler(messagingService), requireUser, sendLimit)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, reportLimit)
	alertsfeature.RegisterRoutes(r, alertsfeature.NewHandler(alertsService), requireUser)

	r.GET("/ws", hub.Handler(func(ctx context.Context, token string) (string, bool) {
		userID, sessionID, err := middleware.ParseAccessToken(a.jwtSecret, token)
		if err != nil || !authService.ValidateSession(ctx, userID, sessionID) {
			return "", false
		}
		return userID, true
	}, a.allowedOrigins))
	return r, nil
}
