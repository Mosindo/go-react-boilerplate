package main

import (
	"context"
	"time"

	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	matchingfeature "example.com/api/internal/features/matching"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/signing"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// limits are abuse guards (requests per window), deliberately generous for real people:
// there is no product quota on likes or messages.
type limits struct {
	global      int // per IP per minute, every route
	credentials int // per IP per minute: login, register, password flows
	swipes      int // per user per minute
	messages    int // per user per minute
	uploads     int // per user per 10 minutes
	reports     int // per user per hour
}

func defaultLimits() limits {
	return limits{global: 600, credentials: 20, swipes: 120, messages: 60, uploads: 30, reports: 10}
}

type deps struct {
	pool           *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	uploadDir      string
	resetURL       string
	mailer         mailer.Mailer
	syncMail       bool
	limits         limits
}

func newDeps(cfg config.Config, pool *pgxpool.Pool) deps {
	var m mailer.Mailer = mailer.NewLog()
	if cfg.SMTP.Enabled() {
		m = mailer.NewSMTP(mailer.SMTPConfig(cfg.SMTP))
	}
	return deps{
		pool:           pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		uploadDir:      cfg.UploadDir,
		resetURL:       cfg.PasswordResetURL,
		mailer:         m,
		limits:         defaultLimits(),
	}
}

// accountEraser cleans up what lives outside the database when an account is deleted.
type accountEraser struct {
	photos *photosfeature.Service
	hub    *realtime.Hub
}

func (e accountEraser) Prepare(ctx context.Context, userID string) (func(), error) {
	finishPhotos, err := e.photos.Prepare(ctx, userID)
	if err != nil {
		return nil, err
	}
	return func() {
		finishPhotos()
		e.hub.DisconnectUser(userID)
	}, nil
}

func newRouter(d deps) (*gin.Engine, error) {
	r := gin.New()
	r.MaxMultipartMemory = 10 << 20
	// With no trusted proxy configured Gin ignores X-Forwarded-For, so clients cannot spoof their IP
	// to dodge rate limits. Set TRUSTED_PROXIES when running behind a load balancer.
	if err := r.SetTrustedProxies(d.trustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(d.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.NewRateLimiter(d.limits.global, time.Minute).ByIP("global"))

	signer := signing.New(d.jwtSecret)
	hub := realtime.NewHub()

	storage, err := photosfeature.NewLocalStorage(d.uploadDir)
	if err != nil {
		return nil, err
	}
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(d.pool), storage, signer)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(d.pool, signer))

	authOpts := []authfeature.Option{
		authfeature.WithMailer(d.mailer, d.resetURL),
		authfeature.WithAccountEraser(accountEraser{photos: photosService, hub: hub}),
	}
	if d.syncMail {
		authOpts = append(authOpts, authfeature.WithSyncMail())
	}
	authService := authfeature.NewService(authfeature.NewPGRepository(d.pool), d.jwtSecret, authOpts...)

	requireUser := middleware.RequireUser(d.jwtSecret, authService)
	credentialLimit := middleware.NewRateLimiter(d.limits.credentials, time.Minute).ByIP("credentials")
	swipeLimit := middleware.NewRateLimiter(d.limits.swipes, time.Minute).ByUser("swipes")
	messageLimit := middleware.NewRateLimiter(d.limits.messages, time.Minute).ByUser("messages")
	uploadLimit := middleware.NewRateLimiter(d.limits.uploads, 10*time.Minute).ByUser("uploads")
	reportLimit := middleware.NewRateLimiter(d.limits.reports, time.Hour).ByUser("reports")

	r.GET("/health", platformhandlers.NewHealthHandler(d.pool))
	r.GET("/ws", hub.Handler(func(token string) (string, time.Time, error) {
		claims, err := middleware.Authenticate(context.Background(), d.jwtSecret, authService, token)
		return claims.UserID, claims.ExpiresAt, err
	}))

	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, credentialLimit)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, uploadLimit)
	matchingfeature.RegisterRoutes(r,
		matchingfeature.NewHandler(matchingfeature.NewService(matchingfeature.NewPGRepository(d.pool), profilesService, hub)),
		requireUser, swipeLimit)
	chatfeature.RegisterRoutes(r,
		chatfeature.NewHandler(chatfeature.NewService(chatfeature.NewPGRepository(d.pool), profilesService, hub)),
		requireUser, messageLimit)
	notificationsfeature.RegisterRoutes(r,
		notificationsfeature.NewHandler(notificationsfeature.NewService(notificationsfeature.NewPGRepository(d.pool))), requireUser)
	safetyfeature.RegisterRoutes(r,
		safetyfeature.NewHandler(safetyfeature.NewService(safetyfeature.NewPGRepository(d.pool), profilesService, hub)),
		requireUser, reportLimit)
	return r, nil
}
