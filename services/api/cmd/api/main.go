package main

import (
	"context"
	"errors"
	"fmt"
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
	usersfeature "example.com/api/internal/features/users"
	"example.com/api/internal/platform/config"
	"example.com/api/internal/platform/db"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type app struct {
	dbPool         *pgxpool.Pool
	jwtSecret      []byte
	allowedOrigins []string
	trustedProxies []string
	store          storage.Store
	mailer         mailer.Mailer
	appName        string
	rateLimit      bool
	hub            *realtime.Hub
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

	if len(os.Args) > 1 {
		if err := runCommand(ctx, pool, os.Args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	}

	store, err := storage.NewLocalStore(cfg.UploadDir)
	if err != nil {
		log.Fatal(err)
	}

	a := &app{
		dbPool:         pool,
		jwtSecret:      []byte(cfg.JWTSecret),
		allowedOrigins: cfg.AllowedOrigins,
		trustedProxies: cfg.TrustedProxies,
		store:          store,
		mailer: mailer.New(mailer.Config{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom,
		}, !cfg.IsProduction()),
		appName:   cfg.AppName,
		rateLimit: cfg.RateLimit,
		hub:       realtime.NewHub(),
	}

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
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
		log.Printf("api listening on %s (%s)", srv.Addr, cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	<-stop
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

// runCommand implements the maintenance CLI: `api promote-admin <email>`.
func runCommand(ctx context.Context, pool *pgxpool.Pool, args []string) error {
	switch args[0] {
	case "promote-admin":
		if len(args) != 2 {
			return fmt.Errorf("usage: api promote-admin <email>")
		}
		tag, err := pool.Exec(ctx, `UPDATE users SET role = 'admin' WHERE lower(email) = lower($1)`, args[1])
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("no user with email %q", args[1])
		}
		log.Printf("%s is now an admin", args[1])
		return nil
	default:
		return fmt.Errorf("unknown command %q (available: promote-admin)", args[0])
	}
}

type limits struct {
	global, auth, sensitive, swipe, send, upload, report gin.HandlerFunc
}

func newLimits(enabled bool) limits {
	if !enabled {
		noop := func(c *gin.Context) { c.Next() }
		return limits{noop, noop, noop, noop, noop, noop, noop}
	}
	return limits{
		global:    middleware.RateLimitByIP(middleware.NewLimiter(600, time.Minute)),
		auth:      middleware.RateLimitByIP(middleware.NewLimiter(10, time.Minute)),
		sensitive: middleware.RateLimitByUser(middleware.NewLimiter(5, time.Minute)),
		swipe:     middleware.RateLimitByUser(middleware.NewLimiter(120, time.Minute)),
		send:      middleware.RateLimitByUser(middleware.NewLimiter(60, time.Minute)),
		upload:    middleware.RateLimitByUser(middleware.NewLimiter(20, time.Minute)),
		report:    middleware.RateLimitByUser(middleware.NewLimiter(10, time.Hour)),
	}
}

func setupRouter(a *app) *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies(a.trustedProxies) // nil: trust none, so X-Forwarded-For cannot spoof rate limits
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(a.allowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	lim := newLimits(a.rateLimit)
	r.Use(lim.global)
	r.Use(middleware.LimitJSONBody(1 << 20))

	requireUser := middleware.RequireUser(a.jwtSecret, middleware.NewPGSessionChecker(a.dbPool))

	// Repositories and services (handler -> service -> repository).
	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(a.dbPool), a.hub)
	profilesService := profilesfeature.NewService(profilesfeature.NewPGRepository(a.dbPool))

	authService := authfeature.NewService(authfeature.NewPGRepository(a.dbPool), a.jwtSecret)
	authService.SetMailer(a.mailer, a.appName)
	photosService := photosfeature.NewService(photosfeature.NewPGRepository(a.dbPool), a.store, profilesService)
	matchingService := matchingfeature.NewService(matchingfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, a.hub)
	chatService := chatfeature.NewService(chatfeature.NewPGRepository(a.dbPool), profilesService, notificationsService, a.hub)
	safetyService := safetyfeature.NewService(safetyfeature.NewPGRepository(a.dbPool), a.hub)
	usersService := usersfeature.NewService(usersfeature.NewPGRepository(a.dbPool), a.store, a.hub)

	r.GET("/health", platformhandlers.NewHealthHandler(a.dbPool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, lim.auth)
	usersfeature.RegisterRoutes(r, usersfeature.NewHandler(usersService), requireUser, lim.sensitive)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesService), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosService), requireUser, lim.upload)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingService), requireUser, lim.swipe)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatService), requireUser, lim.send)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetyService), requireUser, middleware.RequireAdmin(), lim.report)

	// Realtime: exchange the access token for a 60 s ticket, then open the socket with it.
	r.POST("/ws/ticket", requireUser, func(c *gin.Context) {
		ticket, err := realtime.IssueTicket(a.jwtSecret, c.GetString("userID"))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not issue ticket"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ticket": ticket, "expiresIn": 60})
	})
	r.GET("/ws", a.hub.Handler(a.jwtSecret, a.allowedOrigins))
	return r
}
