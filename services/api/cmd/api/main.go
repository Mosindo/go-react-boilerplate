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
	cfg    config.Config
	dbPool *pgxpool.Pool
	mailer mailer.Mailer
	hub    *realtime.Hub
	// authRateLimit overrides the per-IP auth budget per minute (tests).
	authRateLimit int
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

	a := &app{cfg: cfg, dbPool: pool, hub: realtime.NewHub()}
	if cfg.SMTPHost != "" {
		a.mailer = mailer.SMTPMailer{Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.MailFrom}
	} else {
		a.mailer = mailer.LogMailer{}
	}

	router, err := setupRouter(a)
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func setupRouter(a *app) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(a.cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.MaxMultipartMemory = 8 << 20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.LimitJSONBody(64 << 10))

	storage, err := photosfeature.NewStorage(a.cfg.UploadDir)
	if err != nil {
		return nil, err
	}

	authRepo := authfeature.NewPGRepository(a.dbPool)
	authService := authfeature.NewService(authRepo, []byte(a.cfg.JWTSecret), authfeature.Options{
		Mailer:      a.mailer,
		ResetURL:    a.cfg.PasswordResetURL,
		Files:       storage,
		Connections: a.hub,
	})
	requireUser := middleware.RequireUser([]byte(a.cfg.JWTSecret), authRepo)

	// Budgets are per client IP (auth) or per user (everything else); see README.
	authBudget := 20
	if a.authRateLimit > 0 {
		authBudget = a.authRateLimit
	}
	authLimiter := middleware.NewRateLimiter(authBudget, time.Minute)
	uploadLimiter := middleware.NewRateLimiter(30, time.Hour)
	swipeLimiter := middleware.NewRateLimiter(240, time.Minute)
	messageLimiter := middleware.NewRateLimiter(60, time.Minute)
	reportLimiter := middleware.NewRateLimiter(10, time.Hour)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authLimiter)

	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool))), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), storage)), requireUser, uploadLimiter)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), a.hub)), requireUser, swipeLimiter)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), a.hub)), requireUser, messageLimiter)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool))), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), a.hub)), requireUser, reportLimiter)

	r.GET("/ws", realtime.Handler(a.hub, []byte(a.cfg.JWTSecret), authRepo, a.cfg.AllowedOrigins))
	return r, nil
}
