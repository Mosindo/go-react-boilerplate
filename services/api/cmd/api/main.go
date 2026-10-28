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
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// app holds the shared dependencies; one pool and one HTTP server for the whole process.
type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	mailer         mail.Mailer
	blobs          *storage.Local
	hub            *realtime.Hub
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

	blobs, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	var mailer mail.Mailer = mail.LogMailer{}
	if cfg.SMTP.Enabled() {
		mailer = mail.SMTPMailer{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		mailer:         mailer,
		blobs:          blobs,
		hub:            realtime.NewHub(),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
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

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
	// Client IPs (rate limiting, logs) come from forwarding headers only when a trusted proxy is configured.
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

	// Wiring order follows the dependency graph: notifications <- chat <- discovery.
	profilesRepo := profilesfeature.NewPGRepository(a.dbPool)
	profilesService := profilesfeature.NewService(profilesRepo)

	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), a.hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), a.hub, notificationsService)
	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(a.dbPool), profilesService, chatService)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.blobs, profilesService)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool))
	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret, a.mailer, a.blobs)

	requireUser := middleware.RequireUser(a.jwtSecret)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	r.GET("/ws", a.hub.Handler(a.jwtSecret, a.allowedOrigins))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser)
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser)
	return r
}
