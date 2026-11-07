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

type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	appBaseURL     string
	files          *storage.Disk
	mailer         mailer.Mailer
	hub            *realtime.Hub
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
	files, err := storage.NewDisk(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		appBaseURL:     cfg.AppBaseURL,
		files:          files,
		mailer:         mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom),
		hub:            realtime.NewHub(),
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
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.BodyLimit(1 << 20))

	// Per-process rate limiters (see middleware.Limiter).
	authLimiter := middleware.NewLimiter(20, time.Minute)
	swipeLimiter := middleware.NewLimiter(300, time.Minute)
	sendLimiter := middleware.NewLimiter(60, time.Minute)
	uploadLimiter := middleware.NewLimiter(20, time.Minute)
	reportLimiter := middleware.NewLimiter(10, time.Minute)

	authRepo := authfeature.NewPGRepository(a.dbPool)
	authService := authfeature.NewService(authRepo, a.jwtSecret, authfeature.Deps{Mailer: a.mailer, Files: a.files, AppBaseURL: a.appBaseURL})
	requireUser := middleware.RequireUser(a.jwtSecret, authService.SessionActive)

	signer := profilesfeature.NewMediaSigner(a.jwtSecret)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), a.files, signer)
	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(a.dbPool), profilesService, a.hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), a.hub, signer)
	notificationsHandler := notificationsfeature.NewHandler(notificationsfeature.NewRepository(a.dbPool), signer)
	safetyHandler := safetyfeature.NewHandler(safetyfeature.NewRepository(a.dbPool), a.hub, signer)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authLimiter)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService, a.files), requireUser, uploadLimiter)
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser, swipeLimiter)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService),
		chatfeature.NewWSHandler(a.hub, a.jwtSecret, authService.SessionActive, a.allowedOrigins), requireUser, sendLimiter)
	notificationsfeature.RegisterRoutes(r, notificationsHandler, requireUser)
	safetyfeature.RegisterRoutes(r, safetyHandler, requireUser, reportLimiter)
	return r
}
