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
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// app holds the process-level dependencies shared by every feature.
type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	files          storage.Store
	mailer         mailer.Mailer
	hub            *realtime.Hub
	// limitScale multiplies every rate limit (0 means 1). Tests raise it; production leaves it alone.
	limitScale float64
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

	files, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	var m mailer.Mailer = mailer.LogMailer{}
	if cfg.SMTP.Host != "" {
		m = mailer.SMTPMailer{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		files:          files,
		mailer:         m,
		hub:            realtime.NewHub(),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute, // uploads of up to 8 MB on slow mobile links
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
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
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

	// Anti-abuse limits (per process). They are generous for humans and only stop automation.
	scale := a.limitScale
	if scale <= 0 {
		scale = 1
	}
	limit := func(perMinute float64, burst int) *ratelimit.Limiter {
		return ratelimit.New(perMinute*scale, int(float64(burst)*scale))
	}
	limits := struct {
		credentials, recovery, uploads, swipes, messages, reports *ratelimit.Limiter
	}{
		credentials: limit(30, 10),
		recovery:    limit(10, 5),
		uploads:     limit(30, 12),
		swipes:      limit(240, 60),
		messages:    limit(120, 30),
		reports:     limit(20, 5),
	}

	requireUser := middleware.RequireUser(a.jwtSecret)

	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), a.hub)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool))
	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret, a.mailer, a.files)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.files)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, a.hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), profilesService, a.hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), a.hub)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	r.GET("/ws", realtime.Handler(a.hub, a.jwtSecret, a.allowedOrigins))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authfeature.Limits{
		Credentials: limits.credentials,
		Recovery:    limits.recovery,
	})
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, limits.uploads)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, limits.swipes)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, limits.messages)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, limits.reports)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	return r
}
