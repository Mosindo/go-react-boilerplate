package main

import (
	"context"
	"errors"
	"log"
	"net/http"
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
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/media"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	cfg    config.Config
	dbPool *pgxpool.Pool
	store  storage.Store
	mailer mailer.Mailer
	hub    *realtime.Hub
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
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

	store, err := storage.NewLocalStore(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	var m mailer.Mailer = mailer.LogMailer{}
	if cfg.SMTP.Enabled() {
		m = &mailer.SMTPMailer{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
	}

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	a := &app{cfg: cfg, dbPool: pool, store: store, mailer: m, hub: realtime.NewHub()}
	r, err := setupRouter(a)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
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

func setupRouter(a *app) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(a.cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.MaxMultipartMemory = 12 << 20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	secret := []byte(a.cfg.JWTSecret)
	signer := media.NewSigner(secret)
	requireUser := middleware.RequireUser(secret)

	// Abuse protection. Limits are generous for real usage (likes are never capped
	// for product reasons, only throttled against scripted abuse).
	loginLimit := middleware.LimitByIP(middleware.NewRateLimiter(10, 10, time.Minute), "login")
	registerLimit := middleware.LimitByIP(middleware.NewRateLimiter(5, 5, 10*time.Minute), "register")
	resetLimit := middleware.LimitByIP(middleware.NewRateLimiter(5, 5, 15*time.Minute), "reset")
	refreshLimit := middleware.LimitByIP(middleware.NewRateLimiter(30, 30, time.Minute), "refresh")
	swipeLimit := middleware.LimitByUser(middleware.NewRateLimiter(60, 120, time.Minute), "swipe")
	messageLimit := middleware.LimitByUser(middleware.NewRateLimiter(20, 30, time.Minute), "message")
	uploadLimit := middleware.LimitByUser(middleware.NewRateLimiter(10, 20, 10*time.Minute), "upload")
	reportLimit := middleware.LimitByUser(middleware.NewRateLimiter(5, 10, time.Hour), "report")

	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), a.hub)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), signer)
	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), secret, a.mailer, a.store, a.hub)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.store, signer)
	discoveryRepo := discoveryfeature.NewPGRepository(a.dbPool)
	discoveryService := discoveryfeature.NewService(discoveryRepo, profilesService, discoveryfeature.DefaultRanker())
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, a.hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, a.hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), a.hub)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	r.GET("/realtime", realtime.Handler(a.hub, secret, a.cfg.AllowedOrigins, func(userID string) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = discoveryRepo.TouchActivity(ctx, userID)
	}))

	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authfeature.RouteLimits{
		Login: loginLimit, Register: registerLimit, Reset: resetLimit, Refresh: refreshLimit,
	})
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, uploadLimit)
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, swipeLimit)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, messageLimit)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, reportLimit)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ressource introuvable.", "code": "not_found"})
	})
	return r, nil
}
