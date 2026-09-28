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
	matchesfeature "example.com/api/internal/features/matches"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	realtimefeature "example.com/api/internal/features/realtime"
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// jsonBodyLimit caps every JSON request; photo uploads have their own limit.
const jsonBodyLimit = 1 << 20

// rateLimits holds the limiters so tests can swap in tighter or looser ones.
type rateLimits struct {
	Auth    *middleware.Limiter // per IP, all /auth/*
	Swipes  *middleware.Limiter // per user
	Message *middleware.Limiter // per user
	Reports *middleware.Limiter // per user
}

func defaultRateLimits() rateLimits {
	return rateLimits{
		Auth:    middleware.NewLimiter(30, time.Minute),
		Swipes:  middleware.NewLimiter(120, time.Minute),
		Message: middleware.NewLimiter(30, time.Minute),
		Reports: middleware.NewLimiter(10, time.Hour),
	}
}

type deps struct {
	Pool           *pgxpool.Pool
	JWTSecret      []byte
	AllowedOrigins []string
	TrustedProxies []string
	Mailer         mailer.Mailer
	Limits         rateLimits
	BcryptCost     int
	Production     bool

	// Filled by setupRouter.
	Hub *realtimefeature.Hub
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

	var m mailer.Mailer = mailer.Log{}
	if cfg.SMTP.Configured() {
		m = mailer.SMTP{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
	}

	d := &deps{
		Pool:           pool,
		JWTSecret:      []byte(cfg.JWTSecret),
		AllowedOrigins: cfg.AllowedOrigins,
		TrustedProxies: cfg.TrustedProxies,
		Mailer:         m,
		Limits:         defaultRateLimits(),
		Production:     cfg.IsProduction(),
	}
	router, err := setupRouter(d)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("api listening on %s (env=%s)", srv.Addr, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		log.Fatal(err)
	case sig := <-stop:
		log.Printf("received %s, shutting down", sig)
	}

	d.Hub.Close() // WebSockets are hijacked; Shutdown does not wait for them.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

// setupRouter wires the single Gin engine. Composition root only: no logic here.
func setupRouter(d *deps) (*gin.Engine, error) {
	if d.Production {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.HandleMethodNotAllowed = true
	if err := r.SetTrustedProxies(d.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.CustomRecovery(func(c *gin.Context, _ any) {
		httpx.Abort(c, http.StatusInternalServerError, httpx.CodeInternal, "internal error")
	}))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(d.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.NoRoute(func(c *gin.Context) { httpx.Abort(c, http.StatusNotFound, httpx.CodeNotFound, "route not found") })
	r.NoMethod(func(c *gin.Context) {
		httpx.Abort(c, http.StatusMethodNotAllowed, httpx.CodeInvalidRequest, "method not allowed")
	})

	if d.Hub == nil {
		d.Hub = realtimefeature.NewHub()
	}
	if d.Mailer == nil {
		d.Mailer = mailer.Log{}
	}
	if d.Limits.Auth == nil {
		d.Limits = defaultRateLimits()
	}

	sessions := db.NewSessionChecker(d.Pool)
	requireUser := middleware.RequireUser(d.JWTSecret, sessions)
	publisher := d.Hub

	api := r.Group("", middleware.BodyLimit(jsonBodyLimit))
	upload := r.Group("", middleware.BodyLimit(photosfeature.MaxRequestBytes))

	r.GET("/health", platformhandlers.NewHealthHandler(d.Pool))

	authService := authfeature.NewService(authfeature.NewPGRepository(d.Pool), d.JWTSecret, d.Mailer, d.BcryptCost)
	authfeature.RegisterRoutes(api, authfeature.NewHandler(authService), requireUser, middleware.RateLimitByIP(d.Limits.Auth))

	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(d.Pool))
	profilesfeature.RegisterRoutes(api, profilesfeature.NewHandler(profilesService), requireUser)

	photosService := photosfeature.NewService(photosfeature.NewPGRepository(d.Pool))
	photosfeature.RegisterRoutes(api, upload, photosfeature.NewHandler(photosService), requireUser)

	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(d.Pool), discoveryfeature.DefaultRanker{}, publisher)
	discoveryfeature.RegisterRoutes(api, discoveryfeature.NewHandler(discoveryService), requireUser, middleware.RateLimitByUser(d.Limits.Swipes))

	matchesService := matchesfeature.NewService(matchesfeature.NewPGRepository(d.Pool), publisher)
	matchesfeature.RegisterRoutes(api, matchesfeature.NewHandler(matchesService), requireUser)

	chatService := chatfeature.NewService(chatfeature.NewPGRepository(d.Pool), publisher)
	chatfeature.RegisterRoutes(api, chatfeature.NewHandler(chatService), requireUser, middleware.RateLimitByUser(d.Limits.Message))

	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(d.Pool))
	safetyfeature.RegisterRoutes(api, safetyfeature.NewHandler(safetyService), requireUser, middleware.RateLimitByUser(d.Limits.Reports))

	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(d.Pool))
	notificationsfeature.RegisterRoutes(api, notificationsfeature.NewHandler(notificationsService), requireUser)

	realtimefeature.RegisterRoutes(r, realtimefeature.NewHandler(d.Hub, d.JWTSecret, sessions, d.AllowedOrigins))
	return r, nil
}
