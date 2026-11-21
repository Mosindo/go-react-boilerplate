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
	commentsfeature "example.com/api/internal/features/comments"
	discoveryfeature "example.com/api/internal/features/discovery"
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
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	mediaSecret    []byte
	store          storage.Storage
	mailer         mailer.Mailer
	hub            *realtime.Hub
	allowedOrigins []string
	trustedProxies []string
	// communityModules registers the generic boilerplate modules.
	communityModules bool
	// rateLimits enables abuse protection. Tests construct the app without it.
	rateLimits bool
	// rateLimitScale multiplies every limit (development uses a generous scale
	// so local end-to-end runs are not throttled). Zero means 1.
	rateLimitScale int
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

	store, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	var mail mailer.Mailer
	switch {
	case cfg.SMTP.Enabled():
		mail = mailer.SMTP{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
	case !cfg.IsProduction():
		log.Printf(`{"event":"mailer_dev_log","message":"SMTP not configured: recovery codes are printed to the log (development only)"}`)
		mail = mailer.DevLog{}
	default:
		log.Printf(`{"event":"mailer_disabled","message":"SMTP not configured: password recovery emails cannot be sent"}`)
		mail = mailer.Disabled{}
	}

	mediaSecret := []byte(cfg.MediaSigningSecret)
	if len(mediaSecret) == 0 {
		mediaSecret = []byte(cfg.JWTSecret)
	}

	a := &app{
		dbPool:           pool,
		jwtSecret:        []byte(cfg.JWTSecret),
		mediaSecret:      mediaSecret,
		store:            store,
		mailer:           mail,
		hub:              realtime.NewHub(),
		allowedOrigins:   cfg.AllowedOrigins,
		trustedProxies:   cfg.TrustedProxies,
		communityModules: cfg.EnableCommunityModules,
		rateLimits:       true,
		rateLimitScale:   1,
	}
	if !cfg.IsProduction() {
		a.rateLimitScale = 20
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
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
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func (a *app) limiter(limit int, window time.Duration, key func(*gin.Context) string) gin.HandlerFunc {
	if !a.rateLimits {
		return middleware.RateLimit(nil, key)
	}
	scale := a.rateLimitScale
	if scale < 1 {
		scale = 1
	}
	return middleware.RateLimit(middleware.NewRateLimiter(limit*scale, window), key)
}

func setupRouter(a *app) *gin.Engine {
	if a.hub == nil {
		a.hub = realtime.NewHub()
	}
	if a.mailer == nil {
		a.mailer = mailer.Disabled{}
	}
	if len(a.mediaSecret) == 0 {
		a.mediaSecret = a.jwtSecret
	}

	r := gin.New()
	// Only trust X-Forwarded-For from configured proxies; otherwise clients
	// could spoof their IP and bypass per-IP rate limits.
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
	r.Use(middleware.MaxJSONBody(1<<20, "/me/photos"))
	r.MaxMultipartMemory = 12 << 20

	signer := media.NewSigner(a.mediaSecret)

	authRepo := authfeature.NewPGRepository(a.dbPool)
	authService := authfeature.NewService(authRepo, a.jwtSecret, authfeature.Options{
		Mailer:    a.mailer,
		Storage:   a.store,
		Publisher: a.hub,
		Sessions:  a.hub,
	})
	authHandler := authfeature.NewHandler(authService)
	requireUser := middleware.RequireUser(a.jwtSecret, authService.SessionActive)

	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), a.hub)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), signer)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.store, signer, profilesService)
	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(a.dbPool), discoveryfeature.NewScoreRanker(), profilesService)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, a.hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, a.hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), profilesService, a.hub)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	r.GET("/realtime", a.hub.Handler(func(token string) (string, string, time.Time, error) {
		identity, err := middleware.ParseAccessToken(a.jwtSecret, token)
		if err != nil {
			return "", "", time.Time{}, err
		}
		active, err := authService.SessionActive(context.Background(), identity.SessionID, identity.UserID)
		if err != nil || !active {
			return "", "", time.Time{}, middleware.ErrInvalidToken
		}
		return identity.UserID, identity.SessionID, identity.ExpiresAt, nil
	}))

	authfeature.RegisterRoutes(r, authHandler, requireUser, authfeature.RouteLimits{
		Register: a.limiter(20, time.Hour, middleware.KeyByIP),
		Login:    a.limiter(30, 5*time.Minute, middleware.KeyByIP),
		Recovery: a.limiter(8, 15*time.Minute, middleware.KeyByIP),
		Delete:   a.limiter(5, 15*time.Minute, middleware.KeyByUser),
	})
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, a.limiter(30, time.Hour, middleware.KeyByUser))
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, a.limiter(1000, time.Hour, middleware.KeyByUser))
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, a.limiter(60, time.Minute, middleware.KeyByUser))
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, a.limiter(20, time.Hour, middleware.KeyByUser))
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)

	// Generic platform modules kept from the boilerplate. The dating client
	// does not use them, so they are only exposed when explicitly enabled.
	if !a.communityModules {
		return r
	}
	usersfeature.RegisterRoutes(r, usersfeature.NewHandler(usersfeature.NewService(usersfeature.NewPGRepository(a.dbPool))), requireUser)
	postsfeature.RegisterRoutes(r, postsfeature.NewHandler(postsfeature.NewService(postsfeature.NewPGRepository(a.dbPool))), requireUser)
	commentsfeature.RegisterRoutes(r, commentsfeature.NewHandler(commentsfeature.NewService(commentsfeature.NewPGRepository(a.dbPool))), requireUser)
	filesfeature.RegisterRoutes(r, filesfeature.NewHandler(filesfeature.NewService(filesfeature.NewPGRepository(a.dbPool))), requireUser)
	return r
}
