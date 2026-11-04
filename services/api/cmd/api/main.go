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
	discoveryfeature "example.com/api/internal/features/discovery"
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
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	store          *media.Store
	mailer         mailer.Mailer
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

	store, err := media.NewStore(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	var m mailer.Mailer = mailer.LogMailer{}
	if cfg.SMTPAddr != "" {
		m = mailer.SMTPMailer{Addr: cfg.SMTPAddr, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom}
	} else {
		log.Print("SMTP_ADDR not set: password recovery codes are written to the log (development only)")
	}

	a := &app{
		dbPool: pool, jwtSecret: []byte(cfg.JWTSecret), allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies, store: store, mailer: m,
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
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
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	// Without explicit trusted proxies X-Forwarded-For is ignored, so clients
	// cannot spoof their IP to dodge rate limits.
	if err := r.SetTrustedProxies(a.trustedProxies); err != nil {
		log.Fatal(err)
	}
	r.MaxMultipartMemory = 8 << 20
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	hub := realtime.NewHub()
	signer := media.NewSigner(a.jwtSecret)

	authRepo := authfeature.NewPGRepository(a.dbPool)
	authService := authfeature.NewService(authRepo, a.jwtSecret, a.mailer)
	requireUser := middleware.RequireUser(a.jwtSecret, authService)

	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), hub)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool), signer)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.store, signer)
	discoveryService := discoveryfeature.NewService(discoveryfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), signer, hub, notificationsService)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), signer)
	accountService := accountfeature.NewService(accountfeature.NewPGRepository(a.dbPool), a.store)

	// Anti-abuse throttles. They protect the service, they are not product limits.
	strict := middleware.RateLimitByIP(middleware.NewLimiter(20, 10))
	uploadLimit := middleware.RateLimitByUser(middleware.NewLimiter(30, 10))
	swipeLimit := middleware.RateLimitByUser(middleware.NewLimiter(180, 60))
	sendLimit := middleware.RateLimitByUser(middleware.NewLimiter(90, 30))
	reportLimit := middleware.RateLimitByUser(middleware.NewLimiter(10, 5))

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, strict)
	accountfeature.RegisterRoutes(r, accountfeature.NewHandler(accountService), requireUser, strict)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, uploadLimit)
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoveryService), requireUser, swipeLimit)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), chatfeature.NewWSHandler(hub, a.jwtSecret, authService), requireUser, sendLimit)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, reportLimit)
	return r
}
