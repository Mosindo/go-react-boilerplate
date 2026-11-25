// Package app is the composition root: it wires platform services and
// features into a single Gin engine.
package app

import (
	"time"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/features/chat"
	"example.com/api/internal/features/matching"
	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/photos"
	"example.com/api/internal/features/profiles"
	"example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"example.com/api/internal/platform/token"
	"example.com/api/internal/platform/urlsign"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Limits groups the rate limits applied to abuse-prone endpoints. Tests can
// relax them.
type Limits struct {
	Auth     *middleware.RateLimiter
	Recovery *middleware.RateLimiter
	Upload   *middleware.RateLimiter
	Swipe    *middleware.RateLimiter
	Message  *middleware.RateLimiter
	Report   *middleware.RateLimiter
	Ticket   *middleware.RateLimiter
}

func DefaultLimits() Limits {
	return Limits{
		Auth:     middleware.NewRateLimiter(20, time.Minute),
		Recovery: middleware.NewRateLimiter(5, 15*time.Minute),
		Upload:   middleware.NewRateLimiter(30, 10*time.Minute),
		// Swipes are never limited for product reasons; this only stops bots.
		Swipe:   middleware.NewRateLimiter(300, time.Minute),
		Message: middleware.NewRateLimiter(60, time.Minute),
		Report:  middleware.NewRateLimiter(10, time.Hour),
		Ticket:  middleware.NewRateLimiter(30, time.Minute),
	}
}

// Options customises the application wiring (mainly for tests).
type Options struct {
	Limits Limits
	// Mailer overrides the mailer chosen from configuration when set.
	Mailer mailer.Mailer
}

type App struct {
	Engine *gin.Engine
	Hub    *realtime.Hub
}

func New(cfg config.Config, pool *pgxpool.Pool, opts Options) (*App, error) {
	limits := opts.Limits
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	store, err := storage.NewLocal(cfg.UploadDir)
	if err != nil {
		return nil, err
	}

	var mail mailer.Mailer = mailer.Disabled{}
	switch {
	case cfg.SMTPEnabled():
		mail = mailer.NewSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom)
	case !cfg.IsProduction():
		mail = mailer.Dev{}
	}
	if opts.Mailer != nil {
		mail = opts.Mailer
	}

	secret := []byte(cfg.JWTSecret)
	tokens := token.NewManager(secret, auth.AccessTokenTTL)
	signer := urlsign.New(secret, time.Hour)
	authService := auth.NewService(auth.NewPGRepository(pool), tokens, mail, store)
	hub := realtime.NewHub(authService)
	photoService := photos.NewService(photos.NewPGRepository(pool), store, signer)
	profileService := profiles.NewService(profiles.NewPGRepository(pool), photoService)
	notificationService := notifications.NewService(notifications.NewPGRepository(pool), hub)
	matchingService := matching.NewService(matching.NewPGRepository(pool), profileService, notificationService, hub, !cfg.IsProduction())
	chatService := chat.NewService(chat.NewPGRepository(pool), photoService, hub)
	safetyService := safety.NewService(safety.NewPGRepository(pool), hub)

	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.MaxJSONBody(1 << 20))

	requireUser := middleware.RequireUser(tokens, authService)

	r.GET("/health", platformhandlers.NewHealthHandler(pool))
	auth.RegisterRoutes(r, auth.NewHandler(authService), auth.RouteGuards{
		RequireUser:   requireUser,
		AuthLimit:     limits.Auth.ByIP("auth"),
		RecoveryLimit: limits.Recovery.ByIP("recovery"),
	})
	photos.RegisterRoutes(r, photos.NewHandler(photoService), requireUser, limits.Upload.ByUser("upload"))
	profiles.RegisterRoutes(r, profiles.NewHandler(profileService), requireUser)
	matching.RegisterRoutes(r, matching.NewHandler(matchingService), requireUser, limits.Swipe.ByUser("swipe"))
	chat.RegisterRoutes(r, chat.NewHandler(chatService), requireUser, limits.Message.ByUser("message"))
	notifications.RegisterRoutes(r, notifications.NewHandler(notificationService), requireUser)
	safety.RegisterRoutes(r, safety.NewHandler(safetyService), requireUser, limits.Report.ByUser("report"))
	realtime.RegisterRoutes(r, hub, requireUser, limits.Ticket.ByUser("ticket"))

	return &App{Engine: r, Hub: hub}, nil
}
