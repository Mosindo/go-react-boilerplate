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
	trustedProxies []string
	store          *storage.Store
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

	store, err := storage.New(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	var m mailer.Mailer = mailer.LogMailer{}
	if cfg.IsProduction() {
		// No built-in SMTP: wire a real mailer.Mailer here before relying on password recovery.
		m = mailer.DiscardMailer{}
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		store:          store,
		mailer:         m,
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
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
	if err := r.SetTrustedProxies(a.trustedProxies); err != nil {
		log.Fatal(err)
	}
	r.MaxMultipartMemory = 8 << 20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.RateLimit(middleware.NewRateLimiter(600, time.Minute)))

	hub := realtime.NewHub()

	authRepo := authfeature.NewPGRepository(a.dbPool)
	profilesRepo := profilesfeature.NewPGRepository(a.dbPool)
	profilesService := profilesfeature.NewService(profilesRepo, a.store)
	authService := authfeature.NewService(authRepo, a.jwtSecret).
		WithMailer(a.mailer).
		WithDeletionHooks(profilesService, hub)

	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), hub)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), notificationsService, authActivity{authRepo})
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), hub, notificationsService)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool))

	requireUser := middleware.RequireUser(a.jwtSecret)
	strict := middleware.NewRateLimiter(20, time.Minute)
	uploads := middleware.NewRateLimiter(20, time.Minute)
	swipes := middleware.NewRateLimiter(120, time.Minute)
	messages := middleware.NewRateLimiter(60, time.Minute)
	reports := middleware.NewRateLimiter(10, time.Minute)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, strict)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser, uploads)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, swipes)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), chatfeature.NewSocketHandler(hub, a.jwtSecret, a.allowedOrigins), requireUser, messages)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, reports)
	return r
}

// authActivity adapts the auth repository to matching.Activity.
type authActivity struct{ repo authfeature.Repository }

func (a authActivity) Touch(ctx context.Context, userID string) error {
	return a.repo.TouchActivity(ctx, userID)
}
