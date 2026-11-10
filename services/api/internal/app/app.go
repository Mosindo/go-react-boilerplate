// Package app is the composition root shared by the API binary and the
// HTTP integration tests: it wires repositories, services and routes.
package app

import (
	"context"
	"net/http"
	"strings"

	"example.com/api/internal/features/auth"
	"example.com/api/internal/features/discovery"
	"example.com/api/internal/features/matches"
	"example.com/api/internal/features/notifications"
	"example.com/api/internal/features/photos"
	"example.com/api/internal/features/profiles"
	realtimefeature "example.com/api/internal/features/realtime"
	"example.com/api/internal/features/safety"
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

// Limits configures the anti-abuse rate limiters (events per minute, burst).
// They protect against automation; none of them is a product quota.
type Limits struct {
	GlobalPerIP  [2]int
	Credentials  [2]int
	Refresh      [2]int
	Sensitive    [2]int
	Uploads      [2]int
	Messages     [2]int
	Swipes       [2]int
	Reports      [2]int
	RealtimeAuth [2]int
}

func DefaultLimits() Limits {
	return Limits{
		GlobalPerIP:  [2]int{600, 300},
		Credentials:  [2]int{10, 10},
		Refresh:      [2]int{120, 60},
		Sensitive:    [2]int{5, 5},
		Uploads:      [2]int{20, 10},
		Messages:     [2]int{60, 20},
		Swipes:       [2]int{120, 60},
		Reports:      [2]int{2, 5},
		RealtimeAuth: [2]int{30, 10},
	}
}

type Options struct {
	Pool           *pgxpool.Pool
	JWTSecret      []byte
	Store          storage.Store
	Mailer         mailer.Mailer
	Hub            *realtime.Hub
	AllowedOrigins []string
	TrustedProxies []string
	Limits         Limits
}

const maxJSONBodyBytes = 1 << 20

// accountCleanup removes resources living outside the database after an
// account deletion.
type accountCleanup struct {
	photos *photos.Service
	hub    *realtime.Hub
}

func (a accountCleanup) AccountDeleted(ctx context.Context, userID string, photoKeys []string) {
	a.photos.DeleteFiles(ctx, photoKeys)
	a.hub.Disconnect(userID)
}

func limiter(cfg [2]int, scope string, key ratelimit.KeyFunc) gin.HandlerFunc {
	return ratelimit.Middleware(ratelimit.New(cfg[0], cfg[1]), scope, key)
}

func NewRouter(opts Options) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(opts.TrustedProxies); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS(opts.AllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.RequestStart())
	r.Use(middleware.RequestMetrics())
	r.Use(middleware.RequestLogger())
	r.Use(limiter(opts.Limits.GlobalPerIP, "global", ratelimit.ByIP))
	r.Use(func(c *gin.Context) {
		if !strings.HasPrefix(c.ContentType(), "multipart/") && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBodyBytes)
		}
		c.Next()
	})

	signer := media.NewSigner(opts.JWTSecret)
	requireUser := middleware.RequireUser(opts.JWTSecret)
	byUser := ratelimit.ByUser

	profilesSvc := profiles.NewService(profiles.NewPGRepository(opts.Pool), signer)
	photosSvc := photos.NewService(photos.NewPGRepository(opts.Pool), opts.Store)
	notificationsSvc := notifications.NewService(notifications.NewPGRepository(opts.Pool), opts.Hub)
	matchesSvc := matches.NewService(matches.NewPGRepository(opts.Pool), profilesSvc, notificationsSvc, opts.Hub)
	discoverySvc := discovery.NewService(discovery.NewPGRepository(opts.Pool), profilesSvc, matchesSvc)
	safetySvc := safety.NewService(safety.NewPGRepository(opts.Pool), matchesSvc)
	authSvc := auth.NewService(auth.NewPGRepository(opts.Pool), opts.JWTSecret, opts.Mailer, accountCleanup{photos: photosSvc, hub: opts.Hub})

	r.GET("/health", platformhandlers.NewHealthHandler(opts.Pool))
	auth.RegisterRoutes(r, auth.NewHandler(authSvc), auth.RouteGuards{
		RequireUser: requireUser,
		Credentials: limiter(opts.Limits.Credentials, "credentials", ratelimit.ByIP),
		Refresh:     limiter(opts.Limits.Refresh, "refresh", ratelimit.ByIP),
		Sensitive:   limiter(opts.Limits.Sensitive, "sensitive", byUser),
	})
	profiles.RegisterRoutes(r, profiles.NewHandler(profilesSvc), requireUser)
	photos.RegisterRoutes(r, photos.NewHandler(photosSvc, signer, opts.Store), requireUser, limiter(opts.Limits.Uploads, "uploads", byUser))
	discovery.RegisterRoutes(r, discovery.NewHandler(discoverySvc, profilesSvc), requireUser, limiter(opts.Limits.Swipes, "swipes", byUser))
	matches.RegisterRoutes(r, matches.NewHandler(matchesSvc), requireUser, limiter(opts.Limits.Messages, "messages", byUser))
	safety.RegisterRoutes(r, safety.NewHandler(safetySvc), requireUser, limiter(opts.Limits.Reports, "reports", byUser))
	notifications.RegisterRoutes(r, notifications.NewHandler(notificationsSvc), requireUser)
	realtimefeature.RegisterRoutes(r, realtimefeature.NewHandler(opts.Hub, opts.AllowedOrigins), requireUser, limiter(opts.Limits.RealtimeAuth, "realtime", byUser))

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "route not found", "code": "not_found"})
	})
	return r, nil
}
