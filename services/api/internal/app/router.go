// Package app wires configuration, platform services and feature modules into one Gin engine.
package app

import (
	"net/http"

	authfeature "example.com/api/internal/features/auth"
	chatfeature "example.com/api/internal/features/chat"
	discoveryfeature "example.com/api/internal/features/discovery"
	moderationfeature "example.com/api/internal/features/moderation"
	notificationsfeature "example.com/api/internal/features/notifications"
	photosfeature "example.com/api/internal/features/photos"
	profilesfeature "example.com/api/internal/features/profiles"
	platformhandlers "example.com/api/internal/platform/handlers"
	"example.com/api/internal/platform/mailer"
	"example.com/api/internal/platform/middleware"
	"example.com/api/internal/platform/realtime"
	"example.com/api/internal/platform/storage"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Deps struct {
	Pool           *pgxpool.Pool
	JWTSecret      []byte
	AllowedOrigins []string
	TrustedProxies []string
	Store          storage.Store
	Mailer         mailer.Mailer
}

// Services exposes the pieces other binaries (seed) reuse.
type Services struct {
	Router *gin.Engine
	Photos *photosfeature.Service
	Hub    *realtime.Hub
}

func NewRouter(d Deps) (*Services, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(d.TrustedProxies); err != nil {
		return nil, err
	}
	r.RedirectTrailingSlash = false
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(d.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(limitBodies(1 << 20))

	hub := realtime.NewHub()
	tickets := realtime.NewTickets()

	profilesRepo := profilesfeature.NewRepository(d.Pool)
	profilesSvc := profilesfeature.NewService(profilesRepo)
	photosSvc := photosfeature.NewService(photosfeature.NewRepository(d.Pool), d.Store)
	notificationsSvc := notificationsfeature.NewService(notificationsfeature.NewRepository(d.Pool), hub)
	discoverySvc := discoveryfeature.NewService(discoveryfeature.NewRepository(d.Pool), profilesSvc, notificationsSvc, hub)
	chatSvc := chatfeature.NewService(chatfeature.NewRepository(d.Pool), profilesSvc, notificationsSvc, hub)
	moderationSvc := moderationfeature.NewService(moderationfeature.NewRepository(d.Pool), hub)
	authSvc := authfeature.NewService(authfeature.NewPGRepository(d.Pool), d.JWTSecret, d.Mailer).
		WithAccountDeletion(photosSvc, hub)

	requireUser := middleware.RequireUser(d.JWTSecret)

	r.GET("/health", platformhandlers.NewHealthHandler(d.Pool))
	authfeature.RegisterRoutes(r, authfeature.NewHandler(authSvc), requireUser)
	profilesfeature.RegisterRoutes(r, profilesfeature.NewHandler(profilesSvc), requireUser)
	photosfeature.RegisterRoutes(r, photosfeature.NewHandler(photosSvc), requireUser)
	discoveryfeature.RegisterRoutes(r, discoveryfeature.NewHandler(discoverySvc), requireUser)
	chatfeature.RegisterRoutes(r, chatfeature.NewHandler(chatSvc), requireUser)
	notificationsfeature.RegisterRoutes(r, notificationsfeature.NewHandler(notificationsSvc), requireUser)
	moderationfeature.RegisterRoutes(r, moderationfeature.NewHandler(moderationSvc), requireUser)
	realtime.RegisterRoutes(r, hub, tickets, requireUser, d.AllowedOrigins)

	return &Services{Router: r, Photos: photosSvc, Hub: hub}, nil
}

// limitBodies caps non-upload request bodies; photo routes set their own larger cap.
func limitBodies(max int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil && c.ContentType() != "multipart/form-data" {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, max)
		}
		c.Next()
	}
}
