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
	matchingfeature "example.com/api/internal/features/matching"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	realtimefeature "example.com/api/internal/features/realtime"
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/health"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/tokens"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// app holds the dependencies shared by every feature.
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

	store, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	a := &app{cfg: cfg, dbPool: pool, store: store, mailer: buildMailer(cfg), hub: realtime.NewHub()}
	router, err := setupRouter(a)
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s (env=%s)", server.Addr, cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func buildMailer(cfg config.Config) mailer.Mailer {
	switch {
	case cfg.SMTP.Enabled():
		return mailer.SMTPMailer{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
	case cfg.IsProduction():
		log.Printf("warning: SMTP is not configured; password reset emails will not be delivered")
		return mailer.DiscardMailer{}
	default:
		return mailer.LogMailer{}
	}
}

func setupRouter(a *app) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(a.cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.MaxMultipartMemory = 12 << 20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	secret := []byte(a.cfg.JWTSecret)
	tokenManager := tokens.NewManager(secret)
	signer := media.NewSigner(secret)

	// Repositories & services. Dependencies flow one way:
	// matching -> profiles -> photos; chat -> photos; everything -> notifications/realtime.
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.store, signer)
	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), tokenManager, a.mailer, photosService, a.hub)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), photosService)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), a.hub)
	matchingService := matchingfeature.NewService(
		matchingfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, a.hub,
		matchingfeature.DefaultRanker(), !a.cfg.IsProduction(),
	)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), photosService, a.hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), a.hub)
	realtimeService := realtimefeature.NewService(tokenManager, authService, a.hub)

	// Guards. Limits are anti-abuse ceilings, far above human usage: likes are
	// never artificially capped.
	requireUser := middleware.RequireUser(tokenManager, authService)
	limit := func(perMinute, burst int, key func(*gin.Context) string) gin.HandlerFunc {
		return middleware.RateLimit(ratelimit.New(perMinute, burst), key)
	}

	r.GET("/health", health.NewHandler(a.dbPool))
	r.Use(middleware.MaxBodyBytes(64<<10, "/me/photos"))

	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), authfeature.Guards{
		RequireUser:   requireUser,
		CredentialsRL: limit(10, 10, middleware.KeyByIP),
		RecoveryRL:    limit(5, 5, middleware.KeyByIP),
	})
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), photosfeature.Guards{
		RequireUser: requireUser,
		UploadRL:    limit(20, 10, middleware.KeyByUser),
		UploadBody:  middleware.MaxBodyBytes(photosfeature.MaxUploadBytes + 1<<20),
		JSONBody:    middleware.MaxBodyBytes(64 << 10),
	})
	// Location changes are throttled so a member cannot "walk" a fake
	// position around someone to triangulate them.
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser, limit(6, 6, middleware.KeyByUser))
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, limit(30, 15, middleware.KeyByUser), limit(120, 60, middleware.KeyByUser))
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, limit(40, 20, middleware.KeyByUser))
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, limit(10, 5, middleware.KeyByUser))
	realtimefeature.RegisterRoutes(r, realtimefeature.NewHandler(realtimeService), requireUser, limit(30, 10, middleware.KeyByIP))
	return r, nil
}
