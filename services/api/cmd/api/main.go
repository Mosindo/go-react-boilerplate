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
	"example.com/api/internal/platform/tokens"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
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

	router, err := newRouter(cfg, pool, store, mailer.New(cfg))
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
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

// newRouter is the composition root: it wires repositories, services and
// handlers of every feature onto a single Gin engine.
func newRouter(cfg config.Config, pool *pgxpool.Pool, store storage.Storage, mail mailer.Mailer) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.BodyLimit(64<<10, photosfeature.MaxUploadBytes+(1<<20)))

	secret := []byte(cfg.JWTSecret)
	tm := tokens.NewManager(secret)
	signer := media.NewSigner(secret)
	hub := realtime.NewHub()

	// Abuse protection only: these limits are far above normal human usage and
	// never restrict the free experience (no daily like quota).
	limit := func(perPeriod int, period time.Duration, burst int, key func(*gin.Context) string) gin.HandlerFunc {
		if cfg.RateLimitDisabled {
			return func(c *gin.Context) { c.Next() }
		}
		return middleware.RateLimit(ratelimit.New(perPeriod, period, burst), key)
	}
	strict := limit(10, time.Minute, 10, middleware.KeyByIP)
	standard := limit(60, time.Minute, 30, middleware.KeyByIP)
	swipes := limit(120, time.Minute, 60, middleware.KeyByUser)
	messages := limit(60, time.Minute, 30, middleware.KeyByUser)
	uploads := limit(20, time.Minute, 10, middleware.KeyByUser)
	reports := limit(10, time.Hour, 5, middleware.KeyByUser)
	connects := limit(30, time.Minute, 20, middleware.KeyByIP)

	photosService := photosfeature.NewService(photosfeature.NewPGRepository(pool), store)
	authService := authfeature.NewService(authfeature.NewPGRepository(pool), tm, secret, mail, photosService, hub)
	requireUser := middleware.RequireUser(tm, authService)

	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(pool), signer)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(pool), hub)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(pool), profilesService, notificationsService, hub, matchingfeature.DefaultRanker{})
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(pool), profilesService, notificationsService, hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(pool), hub)
	realtimeService := realtimefeature.NewService(tm)

	r.GET("/health", platformhandlers.NewHealthHandler(pool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), authfeature.RouteGuards{RequireUser: requireUser, Strict: strict, Standard: standard})
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService, profilesService, signer), requireUser, uploads)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, swipes)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, messages)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, reports)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	realtimefeature.RegisterRoutes(r, realtimefeature.NewHandler(realtimeService, hub, authService, cfg.AllowedOrigins), requireUser, connects)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ressource introuvable.", "code": "not_found"})
	})
	return r, nil
}
