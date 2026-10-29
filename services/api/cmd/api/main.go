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

	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	discoveryfeature "example.com/api/internal/features/discovery"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	store          storage.Storage
	mailer         mailer.Mailer
	// rateLimits turns on the per-IP / per-user limiters (always on in production,
	// disabled by tests that fire many requests from one address).
	rateLimits bool
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

	store, err := storage.NewDisk(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	var m mailer.Mailer = mailer.Log{}
	if cfg.SMTP.Enabled() {
		m = mailer.SMTP{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		store:          store,
		mailer:         m,
		rateLimits:     true,
	}
	if !cfg.IsProduction() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
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
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
	if err := r.SetTrustedProxies(a.trustedProxies); err != nil {
		log.Fatal(err)
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	hub := realtime.NewHub()

	authRepo := authfeature.NewPGRepository(a.dbPool)
	authService := authfeature.NewService(authRepo, a.jwtSecret, a.mailer)
	authService.OnAccountDeleted = hub.CloseUser
	authHandler := authfeature.NewHandler(authService)

	profilesRepo := profilesfeature.NewPGRepository(a.dbPool)
	profilesHandler := profilesfeature.NewHandler(profilesfeature.NewService(profilesRepo))

	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.store)
	authService.RemoveFiles = photosService.RemoveFiles
	photosHandler := photosfeature.NewHandler(photosService)

	discoveryHandler := discoveryfeature.NewHandler(discoveryfeature.NewService(discoveryfeature.NewPGRepository(a.dbPool), hub))
	chatHandler := chatfeature.NewHandler(chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), hub))
	safetyHandler := safetyfeature.NewHandler(safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), hub))
	notificationsHandler := notificationsfeature.NewHandler(notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool)))

	requireUser := middleware.RequireUser(a.jwtSecret, authService)

	var authLimits authfeature.Limiters
	var swipeLimit, sendLimit, uploadLimit, reportLimit gin.HandlerFunc
	if a.rateLimits {
		authLimits = authfeature.Limiters{
			Auth:    middleware.NewRateLimiter(30, time.Minute).ByIP(),
			Recover: middleware.NewRateLimiter(10, 15*time.Minute).ByIP(),
		}
		swipeLimit = middleware.NewRateLimiter(240, time.Minute).ByUser()
		sendLimit = middleware.NewRateLimiter(60, time.Minute).ByUser()
		uploadLimit = middleware.NewRateLimiter(20, 10*time.Minute).ByUser()
		reportLimit = middleware.NewRateLimiter(10, time.Hour).ByUser()
	}

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authHandler, requireUser, authLimits)
	profilesfeature.RegisterRoutes(r, profilesHandler, requireUser)
	photosfeature.RegisterRoutes(r, photosHandler, requireUser, uploadLimit)
	discoveryfeature.RegisterRoutes(r, discoveryHandler, requireUser, swipeLimit)
	chatfeature.RegisterRoutes(r, chatHandler, requireUser, sendLimit)
	safetyfeature.RegisterRoutes(r, safetyHandler, requireUser, reportLimit)
	notificationsfeature.RegisterRoutes(r, notificationsHandler, requireUser)

	r.GET("/ws", realtime.Handler(hub, func(ctx context.Context, token string) (string, error) {
		userID, sessionID, err := middleware.ParseAccessToken(a.jwtSecret, token)
		if err != nil {
			return "", err
		}
		active, err := authService.SessionActive(ctx, sessionID, userID)
		if err != nil || !active {
			return "", errors.New("inactive session")
		}
		return userID, nil
	}, a.allowedOrigins))
	return r
}
