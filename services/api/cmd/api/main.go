package main

import (
	"context"
	"errors"
	"log"
	"net/http"
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
	realtimefeature "example.com/api/internal/features/realtime"
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// maxJSONBodyBytes caps every non-upload request body.
const maxJSONBodyBytes = 64 << 10

type app struct {
	cfg    config.Config
	dbPool *pgxpool.Pool
	store  storage.Store
	hub    *realtime.Hub
	mailer mailer.Mailer
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

	var mail mailer.Mailer = mailer.LogMailer{}
	if cfg.SMTP.Enabled() {
		mail = mailer.SMTPMailer{
			Host:     cfg.SMTP.Host,
			Port:     cfg.SMTP.Port,
			Username: cfg.SMTP.Username,
			Password: cfg.SMTP.Password,
			From:     cfg.SMTP.From,
		}
	}

	r, err := setupRouter(&app{cfg: cfg, dbPool: pool, store: store, hub: realtime.NewHub(), mailer: mail})
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		log.Printf("api listening on %s (env=%s)", srv.Addr, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

// isPhotoUpload matches the multipart routes (POST /me/photos and
// PUT /me/photos/:id), which enforce their own 10 MB cap. JSON routes such as
// PUT /me/photos/order keep the global limit.
func isPhotoUpload(r *http.Request) bool {
	path := r.URL.Path
	switch r.Method {
	case http.MethodPost:
		return path == "/me/photos"
	case http.MethodPut:
		return strings.HasPrefix(path, "/me/photos/") && path != "/me/photos/order"
	}
	return false
}

func setupRouter(a *app) (*gin.Engine, error) {
	r := gin.New()
	// Only trust X-Forwarded-For from explicitly configured proxies so rate
	// limits cannot be bypassed by spoofing the header.
	if err := r.SetTrustedProxies(a.cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.MaxBodyBytes(maxJSONBodyBytes, isPhotoUpload))

	secret := []byte(a.cfg.JWTSecret)
	signer := media.NewSigner(secret, time.Hour)

	// Feature wiring: repositories -> services -> handlers.
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), a.hub)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), signer)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.store, signer)
	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), secret, authfeature.Options{
		Mailer:       a.mailer,
		Cleaner:      photosService,
		Disconnector: a.hub,
	})
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, a.hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, a.hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), a.hub)

	requireUser := middleware.RequireUser(secret, authService)

	limit := func(scope string, max int, per time.Duration, key ratelimit.KeyFunc) gin.HandlerFunc {
		return ratelimit.Middleware(ratelimit.New(max, per), scope, key)
	}

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authfeature.Limits{
		Register: limit("register", 10, time.Hour, ratelimit.ByIP),
		Login:    limit("login", 20, 15*time.Minute, ratelimit.ByIP),
		Recovery: limit("recovery", 10, time.Hour, ratelimit.ByIP),
	})
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser,
		limit("upload", 30, time.Hour, ratelimit.ByUserOrIP))
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser,
		limit("swipe", 120, time.Minute, ratelimit.ByUserOrIP))
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser,
		limit("message", 60, time.Minute, ratelimit.ByUserOrIP))
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser,
		limit("report", 20, time.Hour, ratelimit.ByUserOrIP))
	realtimefeature.RegisterRoutes(r, realtimefeature.NewHandler(a.hub, secret, authService, a.cfg.AllowedOrigins))

	return r, nil
}
