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
	matchingfeature "example.com/api/internal/features/matching"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	realtimefeature "example.com/api/internal/features/realtime"
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

// app holds the shared dependencies wired into every feature.
type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	storage        media.Storage
	signer         *media.Signer
	mailer         mailer.Mailer
	hub            *realtime.Hub
	limits         limits
}

// limits are anti-abuse guards, never product quotas.
type limits struct {
	anonymous *ratelimit.Limiter
	swipes    *ratelimit.Limiter
	messages  *ratelimit.Limiter
	uploads   *ratelimit.Limiter
	reports   *ratelimit.Limiter
	tickets   *ratelimit.Limiter
	sensitive *ratelimit.Limiter
	location  *ratelimit.Limiter
}

func defaultLimits() limits {
	return limits{
		anonymous: ratelimit.New(20, time.Minute, 10),
		swipes:    ratelimit.New(120, time.Minute, 60),
		messages:  ratelimit.New(40, time.Minute, 20),
		uploads:   ratelimit.New(30, time.Hour, 10),
		reports:   ratelimit.New(10, time.Hour, 5),
		tickets:   ratelimit.New(30, time.Minute, 10),
		sensitive: ratelimit.New(10, time.Hour, 5),
		location:  ratelimit.New(30, time.Hour, 10),
	}
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

	storage, err := media.NewDiskStorage(cfg.MediaDir)
	if err != nil {
		log.Fatal(err)
	}

	signingSecret := cfg.MediaSigningSecret
	if signingSecret == "" {
		signingSecret = cfg.JWTSecret
	}

	var mail mailer.Mailer
	switch {
	case cfg.SMTP.Enabled():
		mail = mailer.NewSMTPMailer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
	case cfg.IsProduction():
		log.Printf(`{"event":"config_warning","message":"SMTP_HOST is not set: password reset e-mails will not be delivered"}`)
		mail = mailer.DisabledMailer{}
	default:
		mail = mailer.DevMailer{}
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		storage:        storage,
		signer:         media.NewSigner(signingSecret),
		mailer:         mail,
		hub:            realtime.NewHub(),
		limits:         defaultLimits(),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
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
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
	// Only trust X-Forwarded-For from configured proxies, otherwise any client
	// could spoof its IP and bypass per-IP rate limits.
	if err := r.SetTrustedProxies(a.trustedProxies); err != nil {
		log.Fatalf("invalid TRUSTED_PROXIES: %v", err)
	}
	r.MaxMultipartMemory = 12 << 20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret, a.mailer).WithDisconnector(a.hub)
	a.hub.WithSessionChecker(authService)
	requireUser := middleware.RequireUser(a.jwtSecret, authService)
	limit := func(l *ratelimit.Limiter, scope string, key middleware.KeyFunc) gin.HandlerFunc {
		return middleware.RateLimit(l, scope, key)
	}

	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), a.hub)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), a.signer)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.storage, a.signer)
	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(a.dbPool), a.signer)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), a.signer, notificationsService, a.hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), a.signer, notificationsService, a.hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), a.signer, a.hub)
	usersService := usersfeature.NewService(usersfeature.NewPGRepository(a.dbPool), a.storage, a.hub)
	realtimeService := realtimefeature.NewService(a.hub, realtime.NewTicketStore(), authService)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	sensitiveLimit := limit(a.limits.sensitive, "sensitive", middleware.ByUser)
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser,
		limit(a.limits.anonymous, "auth", middleware.ByClientIP), sensitiveLimit)
	usersfeature.RegisterRoutes(r, usersfeature.NewHandler(usersService), requireUser, sensitiveLimit)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser,
		limit(a.limits.location, "location", middleware.ByUser))
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser,
		limit(a.limits.uploads, "uploads", middleware.ByUser))
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser,
		limit(a.limits.swipes, "swipes", middleware.ByUser))
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser,
		limit(a.limits.messages, "messages", middleware.ByUser))
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser,
		limit(a.limits.reports, "reports", middleware.ByUser))
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	realtimefeature.RegisterRoutes(r, realtimefeature.NewHandler(realtimeService), requireUser,
		limit(a.limits.tickets, "tickets", middleware.ByUser))
	return r
}
