package main

import (
	"context"
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
	usersfeature "example.com/api/internal/features/users"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// app holds the shared dependencies used to build the router.
type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	storage        photosfeature.Storage
	mailer         mailer.Mailer
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
	storage, err := photosfeature.NewDiskStorage(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	var m mailer.Mailer = mailer.LogMailer{}
	if cfg.SMTP.Enabled() {
		m = mailer.SMTPMailer{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, User: cfg.SMTP.User, Pass: cfg.SMTP.Pass, From: cfg.SMTP.From}
	}

	a := &app{dbPool: pool, jwtSecret: []byte(cfg.JWTSecret), allowedOrigins: cfg.AllowedOrigins, trustedProxies: cfg.TrustedProxies, storage: storage, mailer: m}
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
	// Never trust X-Forwarded-For unless the operator lists the proxies: otherwise clients could spoof their IP
	// and dodge per-IP rate limits.
	if err := r.SetTrustedProxies(a.trustedProxies); err != nil {
		log.Fatalf("invalid TRUSTED_PROXIES: %v", err)
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	hub := realtime.NewHub()
	requireUser := middleware.RequireUser(a.jwtSecret)

	// Rate limits (per process). Auth limits are per IP, the rest per authenticated user.
	authLimit := ratelimit.New(20, time.Minute).ByIP()
	globalLimit := ratelimit.New(600, time.Minute).ByIP()
	swipeLimit := ratelimit.New(120, time.Minute).ByUser()
	sendLimit := ratelimit.New(60, time.Minute).ByUser()
	uploadLimit := ratelimit.New(20, time.Minute).ByUser()
	reportLimit := ratelimit.New(10, time.Minute).ByUser()
	deleteLimit := ratelimit.New(5, time.Minute).ByUser()
	r.Use(globalLimit)

	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret, a.mailer)
	profilesRepo := profilesfeature.NewPGRepository(a.dbPool)
	profilesService := profilesfeature.NewService(profilesRepo)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.storage)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), profilesRepo, hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewRepository(a.dbPool), hub)
	usersService := usersfeature.NewService(a.dbPool, a.storage)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	r.GET("/ws", hub.Handler(a.jwtSecret))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authLimit)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, uploadLimit)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, swipeLimit)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, sendLimit)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, reportLimit)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsfeature.NewRepository(a.dbPool)), requireUser)
	usersfeature.RegisterRoutes(r, usersfeature.NewHandler(usersService), requireUser, deleteLimit)
	return r
}
