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
	"example.com/api/internal/platform/ratelimit"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Dependencies are the shared infrastructure handed to the router.
type Dependencies struct {
	Config  config.Config
	DB      *pgxpool.Pool
	Storage storage.Storage
	Mailer  mailer.Mailer
	Hub     *realtime.Hub
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

	store, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	var mail mailer.Mailer = mailer.DevLog{}
	if cfg.SMTP.Enabled() {
		mail = mailer.SMTP{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
	}

	router, err := setupRouter(Dependencies{Config: cfg, DB: pool, Storage: store, Mailer: mail, Hub: realtime.NewHub()})
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s (env=%s)", server.Addr, cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func setupRouter(deps Dependencies) (*gin.Engine, error) {
	cfg := deps.Config
	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.MaxMultipartMemory = photosfeature.MaxUploadBytes + 1<<20
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.BodyLimit(1 << 20))

	jwtSecret := []byte(cfg.JWTSecret)
	signer := media.NewSigner(jwtSecret)

	// Rate limits protect against abuse; none of them restricts normal use
	// (likes are unlimited for humans, the swipe limit only stops bots).
	authLimit := ratelimit.Middleware(ratelimit.New(20, time.Minute), "auth", ratelimit.ByIP)
	swipeLimit := ratelimit.Middleware(ratelimit.New(120, time.Minute), "swipe", ratelimit.ByUser)
	messageLimit := ratelimit.Middleware(ratelimit.New(60, time.Minute), "message", ratelimit.ByUser)
	uploadLimit := ratelimit.Middleware(ratelimit.New(20, time.Minute), "upload", ratelimit.ByUser)
	safetyLimit := ratelimit.Middleware(ratelimit.New(30, time.Minute), "safety", ratelimit.ByUser)

	authService := authfeature.NewService(authfeature.NewPGRepository(deps.DB), jwtSecret).WithMailer(deps.Mailer)
	authenticator := middleware.NewAuthenticator(jwtSecret, authService)
	requireUser := authenticator.RequireUser()

	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(deps.DB), deps.Hub)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(deps.DB), signer)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(deps.DB), deps.Storage, signer)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(deps.DB), profilesService, notificationsService, deps.Hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(deps.DB), deps.Hub, notificationsService, profilesService)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(deps.DB), deps.Hub)
	authService.WithAccountCleanup(photosService, deps.Hub)

	r.GET("/health", platformhandlers.NewHealthHandler(deps.DB))
	r.GET("/realtime", realtime.Handler(deps.Hub, authenticator))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authLimit)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, uploadLimit)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, swipeLimit)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, messageLimit)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, safetyLimit)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	return r, nil
}
