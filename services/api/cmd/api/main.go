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
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	jsonBodyLimit      = 1 << 20
	multipartBodyLimit = media.MaxUploadBytes + 256<<10
	photoURLWindow     = 6 * time.Hour
)

// app holds the process-wide dependencies shared by every feature.
type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	storage        media.Storage
	mailer         mailer.Mailer
	hub            *realtime.Hub
	limits         limits
}

type limits struct {
	auth     authfeature.Limiters
	swipes   *middleware.RateLimiter
	messages *middleware.RateLimiter
	reports  *middleware.RateLimiter
	uploads  *middleware.RateLimiter
}

// defaultLimits are anti-abuse ceilings, not product quotas: likes and
// messages stay unlimited for normal human usage.
func defaultLimits() limits {
	return limits{
		auth:     authfeature.DefaultLimiters(),
		swipes:   middleware.NewRateLimiter(120, time.Minute, 60),
		messages: middleware.NewRateLimiter(60, time.Minute, 30),
		reports:  middleware.NewRateLimiter(10, time.Hour, 5),
		uploads:  middleware.NewRateLimiter(30, time.Hour, 12),
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

	storage, err := media.NewLocalStorage(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	var mail mailer.Mailer = mailer.LogMailer{}
	if cfg.SMTP.Host != "" {
		mail = mailer.SMTPMailer{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
	} else {
		log.Printf(`{"event":"mailer","mode":"log","warning":"SMTP_HOST not set: recovery codes are written to the server log (development only)"}`)
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		storage:        storage,
		mailer:         mail,
		hub:            realtime.NewHub(),
		limits:         defaultLimits(),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		// No WriteTimeout: /events is a long-lived stream bounded by the handler.
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}

	go func() {
		log.Printf("api listening on %s", srv.Addr)
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

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
	if err := r.SetTrustedProxies(a.trustedProxies); err != nil {
		log.Fatalf("invalid TRUSTED_PROXIES: %v", err)
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.BodyLimits(jsonBodyLimit, multipartBodyLimit))

	hub := a.hub
	if hub == nil {
		hub = realtime.NewHub()
	}
	signer := media.NewSigner(a.jwtSecret, photoURLWindow)

	// Domain services.
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), signer)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.storage, signer)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), hub)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), profilesService,
		matchingfeature.DefaultRanker(), notificationsService, hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), profilesService, hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), signer, hub)
	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret,
		authfeature.WithMailer(a.mailer),
		authfeature.WithDeletionHook(func(d authfeature.DeletedAccount) {
			photosService.DeleteFiles(d.PhotoKeys)
			for _, other := range d.FormerMatches {
				hub.Publish(other, "match.removed", map[string]string{"userId": d.UserID})
			}
		}),
	)

	requireUser := middleware.RequireActiveSession(a.jwtSecret, authService.SessionActive)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	r.GET("/events", requireUser, realtime.StreamHandler(hub))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, a.limits.auth)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, a.limits.uploads)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, a.limits.swipes)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, a.limits.reports)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, a.limits.messages)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	usersfeature.RegisterRoutes(r, usersfeature.NewHandler(usersfeature.NewService(usersfeature.NewPGRepository(a.dbPool))), requireUser)

	// Generic community modules kept from the boilerplate (not used by the
	// mobile dating UI, still tested and available for future features).
	postsfeature.RegisterRoutes(r, postsfeature.NewHandler(postsfeature.NewService(postsfeature.NewPGRepository(a.dbPool))), requireUser)
	commentsfeature.RegisterRoutes(r, commentsfeature.NewHandler(commentsfeature.NewService(commentsfeature.NewPGRepository(a.dbPool))), requireUser)
	filesfeature.RegisterRoutes(r, filesfeature.NewHandler(filesfeature.NewService(filesfeature.NewPGRepository(a.dbPool))), requireUser)
	return r
}
