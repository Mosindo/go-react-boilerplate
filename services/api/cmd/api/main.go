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
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// app holds the shared dependencies of the single HTTP server.
type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	store          storage.Store
	mailer         mail.Mailer
	hub            *realtime.Hub
	tickets        *realtime.Tickets
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	if !cfg.IsProduction() {
		gin.SetMode(gin.DebugMode)
	} else {
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

	allowedOrigins := cfg.AllowedOrigins
	if len(allowedOrigins) == 0 && !cfg.IsProduction() {
		// Local development (Expo web on another port). Safe because the API
		// authenticates with bearer tokens only: no cookies are ever sent.
		allowedOrigins = []string{"*"}
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: allowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		store:          store,
		mailer:         buildMailer(cfg),
		hub:            realtime.NewHub(),
		tickets:        realtime.NewTickets(),
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           setupRouter(a),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		log.Printf("api listening on %s (env=%s)", srv.Addr, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func buildMailer(cfg config.Config) mail.Mailer {
	switch {
	case cfg.SMTP.Host != "":
		return mail.NewSMTP(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password, cfg.SMTP.From)
	case !cfg.IsProduction():
		return mail.LogMailer{}
	default:
		return mail.DiscardMailer{}
	}
}

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
	r.MaxMultipartMemory = 8 << 20
	// Without explicit trusted proxies the client IP is the socket address, so
	// X-Forwarded-For cannot be spoofed to dodge the rate limiters.
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

	requireUser := middleware.RequireUser(a.jwtSecret)

	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret, a.mailer)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool))
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.store)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool, nil), a.hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), a.hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), a.hub)
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool))

	// Account deletion: purge stored photos and close live connections before the
	// row (and, by cascade, every dependent record) is deleted.
	authService.OnAccountDeletion(photosService.PurgeUser)
	authService.OnAccountDeletion(func(_ context.Context, userID string) error {
		a.hub.DisconnectUser(userID)
		return nil
	})

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)

	r.POST("/ws/ticket", requireUser, realtime.TicketHandler(a.tickets))
	r.GET("/ws", realtime.ConnectHandler(a.hub, a.tickets))
	return r
}
