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
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// app holds process-wide dependencies shared by all features.
type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	uploadDir      string
	mailer         authfeature.Mailer
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

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		uploadDir:      cfg.UploadDir,
		mailer:         authfeature.LogMailer{},
	}
	if cfg.IsProduction() {
		log.Print("warning: password reset e-mails are only logged; plug a real Mailer for production")
	}

	router, err := setupRouter(a)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
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
	_ = srv.Shutdown(shutdownCtx)
}

func setupRouter(a *app) (*gin.Engine, error) {
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	requireUser := middleware.RequireUser(a.jwtSecret)
	authLimit := ratelimit.New(20, time.Minute).ByIP()
	uploadLimit := ratelimit.New(30, time.Minute).ByUser()
	swipeLimit := ratelimit.New(240, time.Minute).ByUser()
	sendLimit := ratelimit.New(60, time.Minute).ByUser()
	reportLimit := ratelimit.New(10, time.Hour).ByUser()

	storage, err := photosfeature.NewLocalStorage(a.uploadDir)
	if err != nil {
		return nil, err
	}
	hub := realtime.NewHub()

	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret, a.mailer, storage)
	profilesService := profilesfeature.NewService(profilesfeature.NewRepository(a.dbPool))
	photosService := photosfeature.NewService(photosfeature.NewRepository(a.dbPool), storage)
	matchingService := matchingfeature.NewService(matchingfeature.NewRepository(a.dbPool), hub)
	chatService := chatfeature.NewService(chatfeature.NewRepository(a.dbPool), hub)
	notificationsRepo := notificationsfeature.NewRepository(a.dbPool)
	safetyService := safetyfeature.NewService(a.dbPool)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	r.GET("/ws", hub.Handler(func(_ context.Context, token string) (string, bool) {
		userID, _, ok := middleware.VerifyAccessToken(a.jwtSecret, token)
		return userID, ok
	}, a.allowedOrigins))

	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authLimit)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, uploadLimit)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, swipeLimit)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, sendLimit)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsRepo), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, reportLimit)
	return r, nil
}
