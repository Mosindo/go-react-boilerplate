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
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	realtimefeature "example.com/api/internal/features/realtime"
	safetyfeature "example.com/api/internal/features/safety"
	usersfeature "example.com/api/internal/features/users"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/middleware"
	platformrt "example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/signedurl"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// limits are per-window request budgets; tests raise them, production uses defaultLimits.
type limits struct {
	auth              authfeature.Limits
	swipesPerMinute   int
	messagesPerMinute int
	uploadsPerHour    int
	reportsPerHour    int
}

func defaultLimits() limits {
	return limits{auth: authfeature.DefaultLimits(), swipesPerMinute: 120, messagesPerMinute: 60, uploadsPerHour: 40, reportsPerHour: 20}
}

type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	appLinkBase    string
	mailer         mail.Mailer
	limits         limits
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

	var mailer mail.Mailer = mail.LogMailer{}
	if cfg.SMTPHost != "" {
		mailer = mail.SMTPMailer{Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser, Password: cfg.SMTPPassword, From: cfg.SMTPFrom}
	}

	router, err := setupRouter(&app{
		dbPool: pool, jwtSecret: []byte(cfg.JWTSecret), allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies, appLinkBase: cfg.AppLinkBase, mailer: mailer, limits: defaultLimits(),
	})
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
		log.Printf("api listening on %s (%s)", srv.Addr, cfg.Env)
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

func setupRouter(a *app) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(a.trustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.BodyLimit(1<<20, 10<<20))
	r.MaxMultipartMemory = 9 << 20

	hub := platformrt.NewHub()
	signer := signedurl.New(a.jwtSecret)
	requireUser := middleware.RequireUser(a.jwtSecret)

	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret, a.mailer, a.appLinkBase)
	usersService := usersfeature.NewService(usersfeature.NewPGRepository(a.dbPool), signer)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), signer)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), signer, hub)
	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(a.dbPool), usersService, chatService, hub)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool))
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), hub)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, a.limits.auth)
	usersfeature.RegisterRoutes(r, usersfeature.NewHandler(usersService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, a.limits.uploadsPerHour)
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser, a.limits.swipesPerMinute)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, a.limits.messagesPerMinute)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, a.limits.reportsPerHour)
	realtimefeature.RegisterRoutes(r, realtimefeature.NewHandler(hub, a.allowedOrigins), requireUser)
	return r, nil
}
