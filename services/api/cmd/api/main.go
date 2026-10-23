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
	matchingfeature "example.com/api/internal/features/matching"
	moderationfeature "example.com/api/internal/features/moderation"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
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
	authPerMinute  int
	apiPerMinute   int
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 10*time.Second, time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool); err != nil {
		log.Fatal(err)
	}

	var m mailer.Mailer = mailer.LogMailer{}
	if cfg.SMTP.Enabled() {
		m = mailer.SMTPMailer{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
	} else {
		log.Printf("SMTP is not configured: password reset codes are written to the log (development only)")
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		mailer:         m,
		authPerMinute:  cfg.AuthRatePerMinute,
		apiPerMinute:   cfg.APIRatePerMinute,
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
	// Without an explicit proxy list the client IP is the TCP peer, so X-Forwarded-For cannot be spoofed
	// to dodge rate limits.
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
	authHandler := authfeature.NewHandler(authService)

	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool))
	photosHandler := photosfeature.NewHandler(photosService)

	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), photosService)
	profilesHandler := profilesfeature.NewHandler(profilesService)

	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(a.dbPool), photosService)
	discoveryHandler := discoveryfeature.NewHandler(discoveryService)

	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), hub)
	notificationsHandler := notificationsfeature.NewHandler(notificationsService)

	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), notificationsService, hub)
	matchingHandler := matchingfeature.NewHandler(matchingService)

	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), notificationsService, hub)
	chatHandler := chatfeature.NewHandler(chatService)

	moderationService := moderationfeature.NewService(moderationfeature.NewPGRepository(a.dbPool), hub)
	moderationHandler := moderationfeature.NewHandler(moderationService)

	requireUser := middleware.RequireUser(a.jwtSecret, authService.CheckSession)
	publicLimit := middleware.RateLimitByIP(middleware.NewLimiter(orDefault(a.authPerMinute, 20)), "auth")
	userLimit := middleware.RateLimitByUser(middleware.NewLimiter(orDefault(a.apiPerMinute, 600)), "api")
	sendLimit := middleware.RateLimitByUser(middleware.NewLimiter(60), "send")
	uploadLimit := middleware.RateLimitByUser(middleware.NewLimiter(30), "upload")
	authed := []gin.HandlerFunc{requireUser, userLimit}

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	r.GET("/ws", hub.Handler(authService.AuthenticateToken, a.allowedOrigins))

	authfeature.RegisterRoutes(r, authHandler, requireUser, publicLimit)
	profilesfeature.RegisterRoutes(r, profilesHandler, authed...)
	photosfeature.RegisterRoutes(r, photosHandler, uploadLimit, authed...)
	discoveryfeature.RegisterRoutes(r, discoveryHandler, authed...)
	matchingfeature.RegisterRoutes(r, matchingHandler, authed...)
	chatfeature.RegisterRoutes(r, chatHandler, sendLimit, authed...)
	moderationfeature.RegisterRoutes(r, moderationHandler, authed...)
	notificationsfeature.RegisterRoutes(r, notificationsHandler, authed...)
	return r
}

func orDefault(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}
