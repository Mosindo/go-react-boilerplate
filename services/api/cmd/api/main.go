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
	filesfeature "example.com/api/internal/features/files"
	matchingfeature "example.com/api/internal/features/matching"
	notificationsfeature "example.com/api/internal/features/notifications"
	safetyfeature "example.com/api/internal/features/safety"
	usersfeature "example.com/api/internal/features/users"
	"example.com/api/internal/platform/blob"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	store          blob.Store
	mailer         mailer.Mailer
	hub            *realtime.Hub
	limits         rateLimits
}

// rateLimits are anti-abuse guards. They sit far above human usage: the
// product has no like, message or swipe quota.
type rateLimits struct {
	auth   *middleware.Limiter
	upload *middleware.Limiter
	swipe  *middleware.Limiter
	send   *middleware.Limiter
	report *middleware.Limiter
}

func defaultRateLimits() rateLimits {
	return rateLimits{
		auth:   middleware.NewLimiter(30, 10),
		upload: middleware.NewLimiter(30, 10),
		swipe:  middleware.NewLimiter(240, 60),
		send:   middleware.NewLimiter(120, 30),
		report: middleware.NewLimiter(6, 10),
	}
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

	db.StartMaintenance(ctx, pool, log.Printf)

	store, err := blob.NewLocalStore(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		store:          store,
		mailer:         buildMailer(cfg),
		hub:            realtime.NewHub(),
		limits:         defaultRateLimits(),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("api listening on %s (%s)", srv.Addr, cfg.Environment)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func buildMailer(cfg config.Config) mailer.Mailer {
	if cfg.SMTP.Host != "" {
		m, err := mailer.NewSMTP(mailer.SMTPConfig{
			Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From,
		})
		if err != nil {
			log.Fatal(err)
		}
		return m
	}
	if cfg.IsProduction() {
		log.Print("warning: SMTP_HOST is not set; password recovery emails will not be delivered")
		return mailer.DisabledMailer{}
	}
	return mailer.LogMailer{}
}

// accountCleanup releases what other modules own when an account is deleted:
// photo files and live connections.
type accountCleanup struct {
	photos *filesfeature.Service
	hub    *realtime.Hub
}

func (c accountCleanup) BeforeDelete(ctx context.Context, userID string) (func(), error) {
	removeFiles, err := c.photos.BeforeDelete(ctx, userID)
	if err != nil {
		return nil, err
	}
	return func() {
		removeFiles()
		c.hub.DisconnectUser(userID)
	}, nil
}

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
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

	authRepo := authfeature.NewPGRepository(a.dbPool)
	authService := authfeature.NewService(authRepo, a.jwtSecret)
	authService.SetMailer(a.mailer)
	authHandler := authfeature.NewHandler(authService)

	photosService := filesfeature.NewService(filesfeature.NewPGRepository(a.dbPool), a.store, a.jwtSecret)
	authService.SetAccountHooks(accountCleanup{photos: photosService, hub: a.hub})

	usersService := usersfeature.NewService(usersfeature.NewPGRepository(a.dbPool), photosService)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), a.hub)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), usersService, notificationsService, a.hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), photosService, notificationsService, a.hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), photosService, a.hub)

	requireUser := middleware.RequireUser(a.jwtSecret, authService.IsSessionActive)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authHandler, requireUser, a.limits.auth.ByIP())
	usersfeature.RegisterRoutes(r, usersfeature.NewHandler(usersService), requireUser)
	filesfeature.RegisterRoutes(r, filesfeature.NewHandler(photosService), requireUser, a.limits.upload.ByUser())
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, a.limits.swipe.ByUser())
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, a.limits.send.ByUser())
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, a.limits.report.ByUser())

	r.POST("/ws/ticket", requireUser, a.hub.TicketHandler())
	r.GET("/ws", a.hub.ConnectHandler(a.allowedOrigins))
	return r
}
