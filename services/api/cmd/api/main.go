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
	realtimefeature "example.com/api/internal/features/realtime"
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/ratelimit"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	appBaseURL     string
	allowedOrigins []string
	mailer         mailer.Mailer
	// Credentials endpoint limits (0 → defaults: 30/min, burst 10 per IP).
	authPerMinute, authBurst int
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

	var m mailer.Mailer = mailer.LogMailer{}
	if cfg.SMTP.Host != "" {
		m = mailer.SMTPMailer{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		appBaseURL:     cfg.AppBaseURL,
		allowedOrigins: cfg.AllowedOrigins,
		mailer:         m,
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
	_ = srv.Shutdown(shutdownCtx)
}

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
	r.MaxMultipartMemory = 8 << 20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	// Brute-force protection for credentials endpoints; generous per-user
	// ceiling for everything else (swipes are intentionally not capped).
	perMin, burst := a.authPerMinute, a.authBurst
	if perMin == 0 {
		perMin, burst = 30, 10
	}
	authLimit := ratelimit.New(perMin, burst).ByIP()
	apiLimiter := ratelimit.New(600, 200)

	authRepo := authfeature.NewPGRepository(a.dbPool)
	authService := authfeature.NewService(authRepo, a.jwtSecret, a.mailer, a.appBaseURL)
	requireUser := middleware.Chain(middleware.RequireUser(a.jwtSecret, authService), apiLimiter.ByUserOrIP())

	hub := realtimefeature.NewHub()
	notifService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool))
	bridge := &eventBridge{hub: hub, notif: notifService}

	profilesHandler := profilesfeature.NewHandler(profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool)))
	photosHandler := photosfeature.NewHandler(photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool)))
	discoveryHandler := discoveryfeature.NewHandler(discoveryfeature.NewService(discoveryfeature.NewPGRepository(a.dbPool), bridge))
	chatHandler := chatfeature.NewHandler(chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), bridge))
	safetyHandler := safetyfeature.NewHandler(safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), bridge))
	notifHandler := notificationsfeature.NewHandler(notifService)
	realtimeHandler := realtimefeature.NewHandler(hub, a.allowedOrigins)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authLimit)
	profilesfeature.RegisterRoutes(r, profilesHandler, requireUser)
	photosfeature.RegisterRoutes(r, photosHandler, requireUser)
	discoveryfeature.RegisterRoutes(r, discoveryHandler, requireUser)
	chatfeature.RegisterRoutes(r, chatHandler, requireUser)
	safetyfeature.RegisterRoutes(r, safetyHandler, requireUser)
	notificationsfeature.RegisterRoutes(r, notifHandler, requireUser)
	realtimefeature.RegisterRoutes(r, realtimeHandler, requireUser)
	return r
}
