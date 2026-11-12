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
	commentsfeature "example.com/api/internal/features/comments"
	filesfeature "example.com/api/internal/features/files"
	matchingfeature "example.com/api/internal/features/matching"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	postsfeature "example.com/api/internal/features/posts"
	profilesfeature "example.com/api/internal/features/profiles"
	safetyfeature "example.com/api/internal/features/safety"
	usersfeature "example.com/api/internal/features/users"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RateLimits groups the anti-abuse quotas. They are generous on purpose:
// the product never limits likes or messages for normal use.
type RateLimits struct {
	Global   *ratelimit.Memory
	Auth     *ratelimit.Memory
	Swipes   *ratelimit.Memory
	Messages *ratelimit.Memory
	Uploads  *ratelimit.Memory
	Reports  *ratelimit.Memory
	Location *ratelimit.Memory
}

func DefaultRateLimits() RateLimits {
	return RateLimits{
		Global:   ratelimit.NewMemory(600, time.Minute),
		Auth:     ratelimit.NewMemory(20, time.Minute),
		Swipes:   ratelimit.NewMemory(120, time.Minute),
		Messages: ratelimit.NewMemory(60, time.Minute),
		Uploads:  ratelimit.NewMemory(30, time.Hour),
		Reports:  ratelimit.NewMemory(20, time.Hour),
		Location: ratelimit.NewMemory(30, time.Hour),
	}
}

type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	storage        media.Storage
	mailer         mailer.Mailer
	hub            *realtime.Hub
	limits         RateLimits
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

	storage, err := media.NewLocalStorage(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	var mail mailer.Mailer
	switch {
	case cfg.SMTP.Enabled():
		mail = mailer.SMTP{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
	case cfg.IsProduction():
		log.Printf(`{"event":"config_warning","message":"SMTP is not configured: password reset emails are disabled"}`)
		mail = mailer.Disabled{}
	default:
		log.Printf(`{"event":"config_notice","message":"development mode: password reset codes are printed to the server log"}`)
		mail = mailer.DevLog{}
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		storage:        storage,
		mailer:         mail,
		hub:            realtime.NewHub(),
		limits:         DefaultRateLimits(),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
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
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
	// Only trust X-Forwarded-For from explicitly configured proxies, otherwise
	// clients could spoof their IP and bypass per-IP rate limits.
	if err := r.SetTrustedProxies(a.trustedProxies); err != nil {
		log.Fatalf("invalid TRUSTED_PROXIES: %v", err)
	}
	r.MaxMultipartMemory = media.MaxUploadBytes + (1 << 20)
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(ratelimit.Middleware("global", a.limits.Global, ratelimit.ByIP))

	signer := media.NewSigner(a.jwtSecret, time.Hour)

	// Shared services.
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), a.hub)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), signer)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.storage, signer)
	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret, authfeature.Options{
		Mailer: a.mailer,
		Files:  photosService,
	})

	verifier := middleware.NewTokenVerifier(a.jwtSecret, authService.SessionActive)
	requireUser := middleware.RequireUser(verifier)
	limit := func(scope string, l *ratelimit.Memory, key ratelimit.KeyFunc) gin.HandlerFunc {
		return ratelimit.Middleware(scope, l, key)
	}

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	r.GET("/media/:key", media.ServeHandler(a.storage, signer))
	r.GET("/ws", realtime.Handler(a.hub, verifier, a.allowedOrigins))

	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, limit("auth", a.limits.Auth, ratelimit.ByIP))
	usersfeature.RegisterRoutes(r, usersfeature.NewHandler(usersfeature.NewService(usersfeature.NewPGRepository(a.dbPool))), requireUser)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser, limit("location", a.limits.Location, ratelimit.ByUser))
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, limit("uploads", a.limits.Uploads, ratelimit.ByUser))

	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, a.hub)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, limit("swipes", a.limits.Swipes, ratelimit.ByUser))

	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, a.hub)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, limit("messages", a.limits.Messages, ratelimit.ByUser))

	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), profilesService, a.hub)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, limit("reports", a.limits.Reports, ratelimit.ByUser))

	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)

	// Generic platform modules kept from the boilerplate (not surfaced in the dating client).
	postsfeature.RegisterRoutes(r, postsfeature.NewHandler(postsfeature.NewService(postsfeature.NewPGRepository(a.dbPool))), requireUser)
	commentsfeature.RegisterRoutes(r, commentsfeature.NewHandler(commentsfeature.NewService(commentsfeature.NewPGRepository(a.dbPool))), requireUser)
	filesfeature.RegisterRoutes(r, filesfeature.NewHandler(filesfeature.NewService(filesfeature.NewPGRepository(a.dbPool))), requireUser)
	return r
}
