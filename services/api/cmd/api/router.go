package main

import (
	"log"
	"time"

	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	matchingfeature "example.com/api/internal/features/matching"
	moderationfeature "example.com/api/internal/features/moderation"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	"example.com/api/internal/platform/config"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/httpx"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps are the shared resources the HTTP layer is built from.
type Deps struct {
	Pool   *pgxpool.Pool
	Config config.Config
	// Optional overrides (tests).
	Mailer  mailer.Mailer
	Storage storage.Storage
	// Scale multiplies the rate limits; tests raise it to avoid false positives.
	LimitScale int
}

func NewRouter(d Deps) (*gin.Engine, error) {
	cfg := d.Config
	scale := d.LimitScale
	if scale <= 0 {
		scale = 1
	}

	store := d.Storage
	if store == nil {
		local, err := storage.NewLocal(cfg.UploadDir)
		if err != nil {
			return nil, err
		}
		store = local
	}
	mail := d.Mailer
	if mail == nil {
		switch {
		case cfg.SMTP.Enabled():
			mail = mailer.SMTPMailer{Host: cfg.SMTP.Host, Port: cfg.SMTP.Port, Username: cfg.SMTP.Username, Password: cfg.SMTP.Password, From: cfg.SMTP.From}
		case cfg.LogResetCodes:
			log.Println("warning: SMTP is not configured, recovery codes are written to the log (LOG_RESET_CODES=true)")
			mail = mailer.LogMailer{}
		default:
			log.Println("warning: SMTP is not configured, password recovery emails are not delivered")
			mail = mailer.Discard{}
		}
	}

	hub := realtime.NewHub(cfg.AllowedOrigins)

	authRepo := authfeature.NewPGRepository(d.Pool)
	authService := authfeature.NewService(authRepo, []byte(cfg.JWTSecret), mail, store)
	requireUser := middleware.RequireUser([]byte(cfg.JWTSecret), authService)

	notificationsService := notificationsfeature.NewService(notificationsfeature.NewPGRepository(d.Pool), hub)
	profilesRepo := profilesfeature.NewPGRepository(d.Pool)

	// Rate limits: protect expensive or abusable endpoints, never the core product.
	authLimit := middleware.NewRateLimiter(30*scale, time.Minute)
	recoverLimit := middleware.NewRateLimiter(6*scale, 15*time.Minute)
	accountLimit := middleware.NewRateLimiter(10*scale, time.Minute)
	swipeLimit := middleware.NewRateLimiter(300*scale, time.Minute)
	sendLimit := middleware.NewRateLimiter(60*scale, time.Minute)
	uploadLimit := middleware.NewRateLimiter(30*scale, time.Hour)
	reportLimit := middleware.NewRateLimiter(10*scale, time.Hour)
	ticketLimit := middleware.NewRateLimiter(60*scale, time.Minute)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())

	r.GET("/health", platformhandlers.NewHealthHandler(d.Pool))

	authfeature.RegisterRoutes(r, authfeature.NewHandler(authService), requireUser, authfeature.Limiters{
		Auth: authLimit.ByIP(), Recover: recoverLimit.ByIP(), Account: accountLimit.ByUser(),
	})
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesfeature.NewService(profilesRepo)), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosfeature.NewService(photosfeature.NewPGRepository(d.Pool), store)), requireUser, uploadLimit.ByUser())
	matchingfeature.RegisterRoutes(r,
		matchingfeature.NewHandler(matchingfeature.NewService(matchingfeature.NewPGRepository(d.Pool), profilesRepo, notificationsService, hub)),
		requireUser, swipeLimit.ByUser())
	chatfeature.RegisterRoutes(r,
		chatfeature.NewHandler(chatfeature.NewService(chatfeature.NewPGRepository(d.Pool), notificationsService, hub)),
		requireUser, sendLimit.ByUser())
	moderationfeature.RegisterRoutes(r,
		moderationfeature.NewHandler(moderationfeature.NewService(moderationfeature.NewPGRepository(d.Pool), hub)),
		requireUser, reportLimit.ByUser())
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsService), requireUser)

	// Realtime: POST /ws/ticket (authenticated) then GET /ws?ticket=...
	r.POST("/ws/ticket", requireUser, ticketLimit.ByUser(), func(c *gin.Context) {
		ticket, err := hub.IssueTicket(httpx.UserID(c))
		if err != nil {
			httpx.Internal(c, "ws.ticket", err)
			return
		}
		httpx.NoStore(c)
		c.JSON(200, gin.H{"ticket": ticket})
	})
	r.GET("/ws", hub.Handle)

	return r, nil
}
