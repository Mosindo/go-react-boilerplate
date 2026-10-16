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

	accountfeature "example.com/api/internal/features/account"
	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	discoverfeature "example.com/api/internal/features/discover"
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
	"example.com/api/internal/platform/mediaurl"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// app carries the process-wide dependencies handed to the router.
type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	appName        string
	allowedOrigins []string
	trustedProxies []string
	store          storage.Store
	mail           mailer.Mailer
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

	store, err := storage.NewLocalStore(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	var mail mailer.Mailer = mailer.LogMailer{}
	if cfg.SMTPHost != "" {
		mail, err = mailer.NewSMTP(mailer.SMTPConfig{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom,
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		appName:        cfg.AppName,
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		store:          store,
		mail:           mail,
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		log.Print("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()

	log.Printf("api listening on %s (env=%s)", srv.Addr, cfg.Env)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
	// Only trust X-Forwarded-For from proxies the operator named, otherwise clients
	// could spoof their IP and dodge the rate limits.
	if err := r.SetTrustedProxies(a.trustedProxies); err != nil {
		log.Fatal(err)
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	signer := mediaurl.New(a.jwtSecret)
	hub := realtime.NewHub()
	tickets := realtime.NewTickets(time.Minute)

	// Photos and profiles (profiles hydrate every card shown to a user).
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.store)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), signer.URL)

	// Notifications and the event bridge that connects matching/chat to realtime.
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool))
	events := newEventBridge(notificationsService, profilesService, hub)

	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret, a.mail, a.appName)
	discoverService := discoverfeature.NewService(discoverfeature.NewPGRepository(a.dbPool), profilesService)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), profilesService, events)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), profilesService, events)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), profilesService)
	accountService := accountfeature.NewService(a.dbPool, photosService, hub)

	requireUser := middleware.RequireUser(a.jwtSecret)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService, signer), requireUser)
	discoverfeature.RegisterRoutes(r, discoverfeature.NewHandler(discoverService), requireUser)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	accountfeature.RegisterRoutes(r, accountfeature.NewHandler(accountService), requireUser)
	realtimefeature.RegisterRoutes(r, realtimefeature.NewHandler(hub, tickets, a.allowedOrigins), requireUser)
	return r
}
