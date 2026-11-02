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
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.Env == "production" {
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

	router, err := newRouter(pool, cfg)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
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
	_ = srv.Shutdown(shutdownCtx)
}

// newRouter wires every feature (handler -> service -> repository) on one Gin
// engine. It is also used by the integration tests.
func newRouter(pool *pgxpool.Pool, cfg config.Config) (*gin.Engine, error) {
	jwtSecret := []byte(cfg.JWTSecret)

	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.NewRateLimiter(1200, 300).ByIP()) // coarse per-IP flood guard
	r.Use(middleware.ValidateIDParams())

	requireUser := middleware.RequireUser(jwtSecret, db.NewSessionChecker(pool))
	authLimit := middleware.NewRateLimiter(30, 10).ByIP()
	swipeLimit := middleware.NewRateLimiter(600, 120).ByUser("swipe")
	messageLimit := middleware.NewRateLimiter(90, 30).ByUser("message")
	uploadLimit := middleware.NewRateLimiter(30, 10).ByUser("upload")

	hub := realtime.NewHub()

	authService := authfeature.NewService(authfeature.NewPGRepository(pool), jwtSecret).WithMailer(mail.New(cfg))
	notifications := notificationsfeature.NewService(pool, hub)
	profilesService := profilesfeature.NewService(profilesfeature.NewRepository(pool))
	discoveryService := discoveryfeature.NewService(discoveryfeature.NewRepository(pool), notifications, hub)
	chatService := chatfeature.NewService(chatfeature.NewRepository(pool), hub, notifications, chatfeature.NewPGNames(pool))

	r.GET("/health", platformhandlers.NewHealthHandler(pool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authLimit)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosfeature.NewRepository(pool)), requireUser, uploadLimit)
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser, swipeLimit)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, messageLimit)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyfeature.NewService(pool, hub)), requireUser)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notifications), requireUser)
	realtime.RegisterRoutes(r, realtime.NewHandler(hub), requireUser)
	return r, nil
}
