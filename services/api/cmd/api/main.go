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

	router, err := setupRouter(cfg, pool)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
	}

	go func() {
		log.Printf("api listening on %s (env=%s)", srv.Addr, cfg.AppEnv)
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

// setupRouter assemble tous les modules (handler -> service -> repository).
func setupRouter(cfg config.Config, pool *pgxpool.Pool) (*gin.Engine, error) {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
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
	r.Use(middleware.LimitBody(1 << 20))

	secret := []byte(cfg.JWTSecret)
	requireUser := middleware.RequireUser(secret)
	hub := realtime.NewHub()

	storage, err := photosfeature.NewDiskStorage(cfg.UploadDir)
	if err != nil {
		return nil, err
	}
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(pool), storage)

	var mail mailer.Mailer = mailer.Log{}
	if cfg.SMTPHost != "" {
		mail = mailer.SMTP{Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.SMTPFrom}
	}

	authService := authfeature.NewService(authfeature.NewPGRepository(pool), secret, mail, cfg.AppBaseURL, photosService)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(pool))
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(pool), hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(pool), hub)
	moderationService := moderationfeature.NewService(moderationfeature.NewPGRepository(pool), hub)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(pool))

	authLimit := middleware.NewRateLimiter(30, time.Minute).ByIP()
	swipeLimit := middleware.NewRateLimiter(200, time.Minute).ByUser()
	sendLimit := middleware.NewRateLimiter(60, time.Minute).ByUser()
	uploadLimit := middleware.NewRateLimiter(20, time.Minute).ByUser()
	reportLimit := middleware.NewRateLimiter(10, time.Hour).ByUser()

	r.GET("/health", platformhandlers.NewHealthHandler(pool))
	r.GET("/ws", hub.Handler(secret, cfg.AllowedOrigins))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authLimit)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, uploadLimit)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, swipeLimit)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, sendLimit)
	moderationfeature.RegisterRoutes(r, moderationfeature.NewHandler(moderationService), requireUser, reportLimit)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	return r, nil
}
