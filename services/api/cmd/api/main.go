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
	discoverfeature "example.com/api/internal/features/discover"
	matchesfeature "example.com/api/internal/features/matches"
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
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	mailer         mailer.Mailer
	hub            *realtime.Hub
	limiter        *middleware.RateLimiter
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
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
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		mailer:         m,
		hub:            realtime.NewHub(),
		limiter:        middleware.NewRateLimiter(),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
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
	// Trust no proxy headers unless TRUSTED_PROXIES says so: client IPs feed rate limiting.
	if err := r.SetTrustedProxies(a.trustedProxies); err != nil {
		log.Fatalf("invalid TRUSTED_PROXIES: %v", err)
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.BodyLimit(1<<20, "/photos"))

	requireUser := middleware.RequireUser(a.jwtSecret)

	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret, a.mailer, a.hub)

	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool))
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), profilesService)
	discoverService := discoverfeature.NewService(discoverfeature.NewPGRepository(a.dbPool), profilesService)
	matchesService := matchesfeature.NewService(matchesfeature.NewPGRepository(a.dbPool), discoverService, profilesService, a.hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), profilesService, a.hub)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool))
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), a.hub)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, a.limiter)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser, a.limiter)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, a.limiter)
	discoverfeature.RegisterRoutes(r, discoverfeature.NewHandler(discoverService), requireUser, a.limiter)
	matchesfeature.RegisterRoutes(r, matchesfeature.NewHandler(matchesService), requireUser, a.limiter)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), chatfeature.NewSocketHandler(a.hub, a.jwtSecret, a.allowedOrigins), requireUser, a.limiter)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, a.limiter)
	return r
}
