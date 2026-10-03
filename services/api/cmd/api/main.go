package main

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	authfeature "example.com/api/internal/features/auth"
	conversationsfeature "example.com/api/internal/features/conversations"
	matchingfeature "example.com/api/internal/features/matching"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	appBaseURL     string
	allowedOrigins []string
	trustedProxies []string
	uploadsDir     string
	mailer         mail.Mailer
	// rateLimits turns on the abuse-protection limiters; unit/integration tests opt out.
	rateLimits bool
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	pool, err := db.ConnectWithRetry(ctx, cfg.DatabaseURL, 10*time.Second, time.Second)
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
		appBaseURL:     cfg.AppBaseURL,
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		uploadsDir:     cfg.UploadsDir,
		mailer:         mail.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.MailFrom),
		rateLimits:     true,
	}

	r, err := setupRouter(a)
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	log.Printf("api listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func noLimit(c *gin.Context) { c.Next() }

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
	r.Use(middleware.BodyLimit(1<<20, func(c *gin.Context) bool {
		return strings.HasPrefix(c.Request.URL.Path, "/me/photos")
	}))

	limiter := func(perSecond float64, burst int, key func(*gin.Context) string) gin.HandlerFunc {
		if !a.rateLimits {
			return noLimit
		}
		return middleware.RateLimit(perSecond, burst, key)
	}
	if a.rateLimits {
		r.Use(middleware.RateLimit(20, 100, middleware.ByIP))
	}
	authLimiter := limiter(0.2, 10, middleware.ByIP)
	swipeLimiter := limiter(5, 30, middleware.ByUser)
	sendLimiter := limiter(3, 15, middleware.ByUser)
	uploadLimiter := limiter(0.5, 6, middleware.ByUser)
	reportLimiter := limiter(0.1, 5, middleware.ByUser)

	hub := realtime.NewHub()

	notificationsRepo := notificationsfeature.NewPGRepository(a.dbPool)
	notificationsService := notificationsfeature.NewService(notificationsRepo, hub)
	notificationsHandler := notificationsfeature.NewHandler(notificationsService)

	profilesRepo := profilesfeature.NewPGRepository(a.dbPool)
	profilesService := profilesfeature.NewService(profilesRepo)
	profilesHandler := profilesfeature.NewHandler(profilesService)

	storage, err := photosfeature.NewLocalStorage(a.uploadsDir)
	if err != nil {
		return nil, err
	}
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), storage)
	photosHandler := photosfeature.NewHandler(photosService)

	authRepo := authfeature.NewPGRepository(a.dbPool)
	authService := authfeature.NewService(authRepo, a.jwtSecret).WithRecovery(a.mailer, photosService, a.appBaseURL)
	authHandler := authfeature.NewHandler(authService)

	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, hub)
	matchingHandler := matchingfeature.NewHandler(matchingService)

	conversationsService := conversationsfeature.NewService(
		conversationsfeature.NewPGRepository(a.dbPool), hub, notificationsService, hub, profilesService)
	conversationsHandler := conversationsfeature.NewHandler(conversationsService)

	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), hub)
	safetyHandler := safetyfeature.NewHandler(safetyService)

	requireUser := middleware.RequireUser(a.jwtSecret)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	r.GET("/ws", hub.Handler(func(token string) (string, error) {
		identity, err := middleware.ParseAccessToken(a.jwtSecret, token)
		return identity.UserID, err
	}, a.allowedOrigins))
	authfeature.RegisterRoutes(r, authHandler, requireUser, authLimiter)
	profilesfeature.RegisterRoutes(r, profilesHandler, requireUser)
	photosfeature.RegisterRoutes(r, photosHandler, requireUser, uploadLimiter)
	matchingfeature.RegisterRoutes(r, matchingHandler, requireUser, swipeLimiter)
	conversationsfeature.RegisterRoutes(r, conversationsHandler, requireUser, sendLimiter)
	notificationsfeature.RegisterRoutes(r, notificationsHandler, requireUser)
	safetyfeature.RegisterRoutes(r, safetyHandler, requireUser, reportLimiter)
	return r, nil
}
