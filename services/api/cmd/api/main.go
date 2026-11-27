package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	matchingfeature "example.com/api/internal/features/matching"
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

// app holds the shared, process-wide dependencies.
type app struct {
	cfg    config.Config
	dbPool *pgxpool.Pool
	store  storage.Store
	mailer mailer.Mailer
	hub    *realtime.Hub
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

	store, err := storage.NewLocalStore(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	a := &app{
		cfg:    cfg,
		dbPool: pool,
		store:  store,
		mailer: selectMailer(cfg),
		hub:    realtime.NewHub(),
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s (env=%s)", server.Addr, cfg.AppEnv)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)
}

func selectMailer(cfg config.Config) mailer.Mailer {
	if cfg.SMTP.Enabled() {
		return mailer.NewSMTPMailer(mailer.SMTPConfig(cfg.SMTP))
	}
	if cfg.IsProduction() {
		log.Printf(`{"event":"smtp_disabled","detail":"password reset e-mails will not be delivered"}`)
		return mailer.DisabledMailer{}
	}
	return mailer.LogMailer{}
}

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
	// Only trust X-Forwarded-For from configured proxies, otherwise IP-based
	// rate limiting could be bypassed with a spoofed header.
	if err := r.SetTrustedProxies(a.cfg.TrustedProxies); err != nil {
		log.Fatal(err)
	}
	r.MaxMultipartMemory = photosfeature.MaxUploadBytes + 1<<20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	// Coarse per-IP ceiling against floods. Signed media requests are exempt:
	// they are cheap, cached by clients and would otherwise penalise users
	// sharing a carrier-grade NAT address.
	globalLimit := middleware.RateLimit("global", middleware.NewRateLimiter(600, time.Minute, 120), middleware.ByIP)
	r.Use(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/media/") {
			c.Next()
			return
		}
		globalLimit(c)
	})

	jwtSecret := []byte(a.cfg.JWTSecret)
	signer := storage.NewURLSigner(jwtSecret)

	// Features
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), a.hub)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), signer)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.store, signer)
	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), jwtSecret).
		WithMailer(a.mailer).
		WithDeletionHook(photosService).
		WithSessionsRevokedHook(a.hub.DisconnectUser)
	matchingService := matchingfeature.NewService(
		matchingfeature.NewPGRepository(a.dbPool),
		profilesService,
		notificationsService,
		a.hub,
		matchingfeature.Options{IncludeDemo: !a.cfg.IsProduction()},
	)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), a.hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), a.hub)

	requireUser := middleware.RequireUser(jwtSecret, authService)

	limit := func(scope string, perWindow int, window time.Duration, burst int, key middleware.KeyFunc) gin.HandlerFunc {
		return middleware.RateLimit(scope, middleware.NewRateLimiter(perWindow, window, burst), key)
	}

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	r.GET("/ws", realtime.Handler(a.hub, jwtSecret, authService, a.cfg.AllowedOrigins))

	authHandler := authfeature.NewHandler(authService).
		WithLoginAttemptLimiter(middleware.NewRateLimiter(10, 15*time.Minute, 10))
	authfeature.RegisterRoutes(r, authHandler, requireUser, authfeature.Limits{
		Credentials:     limit("auth-credentials", 30, time.Minute, 20, middleware.ByIP),
		Recovery:        limit("auth-recovery", 5, 15*time.Minute, 5, middleware.ByIP),
		Refresh:         limit("auth-refresh", 60, time.Minute, 30, middleware.ByIP),
		AccountPassword: limit("account-password", 5, 15*time.Minute, 5, middleware.ByUser),
	})
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser,
		limit("photo-upload", 30, time.Hour, 10, middleware.ByUser))
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser,
		limit("swipes", 300, time.Minute, 60, middleware.ByUser))
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser,
		limit("messages", 60, time.Minute, 20, middleware.ByUser))
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser,
		limit("reports", 20, time.Hour, 5, middleware.ByUser))
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)

	return r
}
