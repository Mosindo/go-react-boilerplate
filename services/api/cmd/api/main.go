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
	"example.com/api/internal/platform/signing"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	mailer         mailer.Mailer
	hub            *realtime.Hub
	limits         limits
}

// limits are per-process abuse guards, deliberately far above normal human use
// (there is no artificial cap on likes: only automation hits these).
type limits struct {
	credentials *middleware.Limiter
	upload      *middleware.Limiter
	swipe       *middleware.Limiter
	message     *middleware.Limiter
	report      *middleware.Limiter
	location    *middleware.Limiter
}

func defaultLimits(authPer10Min int) limits {
	return limits{
		credentials: middleware.NewLimiter(authPer10Min, 10*time.Minute),
		upload:      middleware.NewLimiter(60, time.Hour),
		swipe:       middleware.NewLimiter(600, time.Minute),
		message:     middleware.NewLimiter(60, time.Minute),
		report:      middleware.NewLimiter(20, time.Hour),
		location:    middleware.NewLimiter(30, time.Hour),
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

	var m mailer.Mailer = mailer.LogMailer{}
	if cfg.SMTP.Host != "" {
		m = mailer.SMTPMailer{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		mailer:         m,
		hub:            realtime.NewHub(),
		limits:         defaultLimits(cfg.AuthRateLimit),
	}

	r, err := setupRouter(a)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func setupRouter(a *app) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(a.trustedProxies); err != nil {
		return nil, err
	}
	r.MaxMultipartMemory = 8 << 20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.BodyLimit(1 << 20))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	signer := signing.New(a.jwtSecret)
	loader := profilesfeature.NewLoader(signer)
	requireUser := middleware.RequireUser(a.jwtSecret)

	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret, a.mailer)
	authService.OnAccountDeleted(a.hub.Disconnect)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), loader, a.dbPool)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), signer)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), loader, a.dbPool, a.hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), loader, a.dbPool, a.hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), signer, a.hub)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), signer)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	r.GET("/ws", a.hub.Handler(func(_ context.Context, token string) (string, error) {
		userID, _, err := middleware.ParseAccessToken(a.jwtSecret, token)
		return userID, err
	}, a.allowedOrigins))

	byUser := func(l *middleware.Limiter) gin.HandlerFunc { return middleware.RateLimitByUser(l) }
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, middleware.RateLimitByIP(a.limits.credentials))
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser, byUser(a.limits.location))
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, byUser(a.limits.upload))
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, byUser(a.limits.swipe))
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, byUser(a.limits.message))
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, byUser(a.limits.report))
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	return r, nil
}
