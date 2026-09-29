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

	accountfeature "example.com/api/internal/features/account"
	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	discoveryfeature "example.com/api/internal/features/discovery"
	matchingfeature "example.com/api/internal/features/matching"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	"example.com/api/internal/platform/deps"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/urlsign"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
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

	store, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(cfg, pool, store),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s (env=%s)", srv.Addr, cfg.Env)
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

func buildMailer(cfg config.Config) mailer.Mailer {
	if cfg.SMTP.Host == "" {
		return mailer.LogMailer{}
	}
	return mailer.SMTPMailer{
		Host: cfg.SMTP.Host, Port: cfg.SMTP.Port,
		Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From,
	}
}

// setupRouter wires the single Gin engine. Tests call it with their own config, pool and storage.
func setupRouter(cfg config.Config, pool *pgxpool.Pool, store storage.Storage) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.LimitBody(1 << 20))

	secret := []byte(cfg.JWTSecret)
	hub := realtime.NewHub()
	notifier := notificationsfeature.NewNotifier(pool, hub)
	requireUser := middleware.RequireUser(secret)

	d := deps.Common{
		Config:        cfg,
		JWTSecret:     secret,
		RequireUser:   requireUser,
		Storage:       store,
		Signer:        urlsign.New(secret),
		Mailer:        buildMailer(cfg),
		Publisher:     hub,
		Notifier:      notifier,
		AuthLimiter:   ratelimit.New(20, time.Minute),
		ActionLimiter: ratelimit.New(300, time.Minute),
	}

	r.GET("/health", platformhandlers.NewHealthHandler(pool))
	authfeature.RegisterRoutes(r, pool, d)
	accountfeature.RegisterRoutes(r, pool, d)
	profilesfeature.RegisterRoutes(r, pool, d)
	photosfeature.RegisterRoutes(r, pool, d)
	discoveryfeature.RegisterRoutes(r, pool, d)
	matchingfeature.RegisterRoutes(r, pool, d)
	chatfeature.RegisterRoutes(r, pool, d)
	notificationsfeature.RegisterRoutes(r, pool, d)
	safetyfeature.RegisterRoutes(r, pool, d)
	realtime.RegisterRoutes(r, realtime.NewHandler(hub, secret, cfg.AllowedOrigins), requireUser)
	return r
}
