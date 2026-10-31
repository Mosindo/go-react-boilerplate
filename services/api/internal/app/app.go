// Package app wires platform services and feature modules into one Gin router.
package app

import (
	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	matchingfeature "example.com/api/internal/features/matching"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	safetyfeature "example.com/api/internal/features/safety"
	"example.com/api/internal/platform/config"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mail"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps are the shared dependencies of the HTTP layer.
type Deps struct {
	DB             *pgxpool.Pool
	JWTSecret      []byte
	AllowedOrigins []string
	// TrustedProxies lists reverse-proxy IPs/CIDRs whose X-Forwarded-For is honored.
	TrustedProxies []string
	Mailer         mail.Mailer
	// Limits can be relaxed in tests; zero values use production defaults.
	AuthLimiter *middleware.RateLimiter
}

// NewRouter builds the single HTTP server router.
func NewRouter(d Deps) *gin.Engine {
	if d.AuthLimiter == nil {
		d.AuthLimiter = middleware.NewRateLimiter(30, 10) // per IP: credential-guessing protection
	}
	if d.Mailer == nil {
		d.Mailer = mail.New(config.SMTPConfig{})
	}
	swipeLimiter := middleware.NewRateLimiter(120, 60)
	messageLimiter := middleware.NewRateLimiter(60, 30)
	uploadLimiter := middleware.NewRateLimiter(30, 12)

	r := gin.New()
	// Without an explicit list ClientIP() ignores spoofable X-Forwarded-For headers.
	if len(d.TrustedProxies) == 0 {
		_ = r.SetTrustedProxies(nil)
	} else if err := r.SetTrustedProxies(d.TrustedProxies); err != nil {
		panic(err)
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(d.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.MaxMultipartMemory = 8 << 20

	hub := realtime.NewHub()
	requireUser := middleware.RequireUser(d.JWTSecret)

	profilesRepo := profilesfeature.NewPGRepository(d.DB)
	profilesSvc := profilesfeature.NewService(profilesRepo)

	r.GET("/health", platformhandlers.NewHealthHandler(d.DB))

	authSvc := authfeature.NewService(authfeature.NewPGRepository(d.DB), d.JWTSecret, d.Mailer)
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authSvc), requireUser, d.AuthLimiter)

	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesSvc), requireUser)

	photosSvc := photosfeature.NewService(photosfeature.NewPGRepository(d.DB), profilesRepo)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosSvc), requireUser, uploadLimiter)

	matchingSvc := matchingfeature.NewService(matchingfeature.NewPGRepository(d.DB), profilesSvc, hub)
	matchingfeature.RegisterRoutes(r, matchingfeature.NewHandler(matchingSvc), requireUser, swipeLimiter)

	chatSvc := chatfeature.NewService(chatfeature.NewPGRepository(d.DB), profilesSvc, hub)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatSvc), chatfeature.NewWSHandler(hub, d.JWTSecret, d.AllowedOrigins), requireUser, messageLimiter)

	safetySvc := safetyfeature.NewService(safetyfeature.NewPGRepository(d.DB))
	safetyfeature.RegisterRoutes(r, safetyfeature.NewHandler(safetySvc), requireUser)

	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsfeature.NewPGRepository(d.DB)), requireUser)
	return r
}
